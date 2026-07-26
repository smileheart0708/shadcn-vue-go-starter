package httpapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/database"
	"main/internal/identity"
	"main/internal/logging"
	"main/internal/setup"
)

type testContext struct {
	handler         http.Handler
	authService     *auth.Service
	identityService *identity.Service
	policiesService *accountpolicies.Service
	setupService    *setup.Service
	logStream       *logging.Stream
	dataDir         string
}

type testSuccessEnvelope[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

type testErrorEnvelope struct {
	Success bool `json:"success"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestInstallSetupOnlyRunsOnceAndCreatesOwnerSession(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)

	stateReq := httptest.NewRequest(http.MethodGet, "/api/install/state", nil)
	stateRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(stateRec, stateReq)

	if stateRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, stateRec.Code)
	}

	var stateEnvelope testSuccessEnvelope[installStateResponse]
	decodeJSONResponse(t, stateRec.Body.Bytes(), &stateEnvelope)
	if stateEnvelope.Data.SetupCompleted {
		t.Fatal("expected fresh install state to be pending")
	}

	session, refreshCookie := performSetup(t, ctx)
	if refreshCookie == nil {
		t.Fatal("expected refresh cookie after setup")
	}
	if session.Viewer.Authorization.Role != authorization.RoleOwner {
		t.Fatalf("expected owner role after setup, got %+v", session.Viewer.Authorization.Role)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/install/setup", strings.NewReader(`{"username":"other","password":"password123"}`))
	secondReq.Header.Set("Content-Type", "application/json")
	secondRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, secondRec.Code, secondRec.Body.String())
	}

	decodeErrorCode(t, secondRec.Body.Bytes(), "setup_completed")
}

func TestSetupRequiredBlocksNormalAuthRoutesUntilInitialized(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"identifier":"owner","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	decodeErrorCode(t, rec.Body.Bytes(), "setup_required")
}

func TestOwnerCanCreateManagedUserByDefault(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)

	rec := createManagedUser(t, ctx, ownerSession.AccessToken, `{"username":"member","password":"member1234"}`)
	if rec.Role != authorization.RoleUser {
		t.Fatalf("expected created user role %q, got %q", authorization.RoleUser, rec.Role)
	}
}

func TestDefaultPoliciesClosePublicRegistrationAndSelfDeletion(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)

	settingsReq := httptest.NewRequest(http.MethodGet, "/api/system/settings", nil)
	settingsReq.Header.Set("Authorization", "Bearer "+ownerSession.AccessToken)
	settingsRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(settingsRec, settingsReq)
	if settingsRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, settingsRec.Code, settingsRec.Body.String())
	}

	var settingsEnvelope testSuccessEnvelope[accountPoliciesResponse]
	decodeJSONResponse(t, settingsRec.Body.Bytes(), &settingsEnvelope)
	if settingsEnvelope.Data.PublicRegistrationEnabled || settingsEnvelope.Data.SelfServiceAccountDeletionEnabled {
		t.Fatalf("expected both policies disabled by default, got %+v", settingsEnvelope.Data)
	}
	if strings.Contains(settingsRec.Body.String(), "authMode") || strings.Contains(settingsRec.Body.String(), "registrationMode") {
		t.Fatalf("expected legacy settings fields to be removed, got %s", settingsRec.Body.String())
	}

	publicConfigReq := httptest.NewRequest(http.MethodGet, "/api/auth/public-config", nil)
	publicConfigRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(publicConfigRec, publicConfigReq)
	if publicConfigRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, publicConfigRec.Code)
	}
	if !strings.Contains(publicConfigRec.Body.String(), `"registrationEnabled":false`) {
		t.Fatalf("expected public registration to be disabled, got %s", publicConfigRec.Body.String())
	}

	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"member","email":"member@example.com","password":"member1234"}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, registerRec.Code, registerRec.Body.String())
	}
	decodeErrorCode(t, registerRec.Body.Bytes(), "registration_disabled")
}

func TestPublicRegistrationToggleControlsRegisterRefreshReplayAndLogoutFlow(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)
	updateSystemSettings(t, ctx, ownerSession.AccessToken, `{"publicRegistrationEnabled":true}`)

	_, memberCookie := registerUser(t, ctx, "member", "member@example.com", "member1234")
	if memberCookie == nil {
		t.Fatal("expected refresh cookie for registered member")
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshReq.AddCookie(memberCookie)
	refreshRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(refreshRec, refreshReq)

	if refreshRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, refreshRec.Code, refreshRec.Body.String())
	}
	var refreshedEnvelope testSuccessEnvelope[sessionResponse]
	decodeJSONResponse(t, refreshRec.Body.Bytes(), &refreshedEnvelope)
	if refreshedEnvelope.Data.Viewer.Authorization.Role != authorization.RoleUser {
		t.Fatalf("expected user role after registration, got %+v", refreshedEnvelope.Data.Viewer.Authorization.Role)
	}
	if strings.Contains(refreshRec.Body.String(), "roleKeys") {
		t.Fatalf("expected viewer authorization to use single role field, got %s", refreshRec.Body.String())
	}

	nextCookie := findCookie(refreshRec.Result().Cookies(), ctx.authServiceCookieName())
	if nextCookie == nil || nextCookie.Value == memberCookie.Value {
		t.Fatal("expected rotated refresh cookie")
	}

	replayReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	replayReq.AddCookie(memberCookie)
	replayRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(replayRec, replayReq)
	if replayRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, replayRec.Code)
	}
	decodeErrorCode(t, replayRec.Body.Bytes(), "invalid_refresh_token")

	_, latestSessionCookie := loginUser(t, ctx, "member", "member1234")
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.AddCookie(latestSessionCookie)
	logoutRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, logoutRec.Code)
	}

	postLogoutRefreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	postLogoutRefreshReq.AddCookie(latestSessionCookie)
	postLogoutRefreshRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(postLogoutRefreshRec, postLogoutRefreshReq)
	if postLogoutRefreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, postLogoutRefreshRec.Code)
	}
}

func TestLogoutIgnoresInvalidRefreshTokenCookie(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	performSetup(t, ctx)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: ctx.authServiceCookieName(), Value: "not-a-refresh-token"})
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestSelfServiceDeletionToggleControlsDeleteFlow(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, ownerCookie := performSetup(t, ctx)
	updateSystemSettings(t, ctx, ownerSession.AccessToken, `{"publicRegistrationEnabled":true}`)

	memberSession, memberCookie := registerUser(t, ctx, "member", "member@example.com", "member1234")

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/account", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+memberSession.AccessToken)
	deleteReq.AddCookie(memberCookie)
	deleteRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, deleteRec.Code)
	}
	decodeErrorCode(t, deleteRec.Body.Bytes(), "account_delete_forbidden")

	updateSystemSettings(t, ctx, ownerSession.AccessToken, `{"selfServiceAccountDeletionEnabled":true}`)

	ownerDeleteReq := httptest.NewRequest(http.MethodDelete, "/api/account", nil)
	ownerDeleteReq.Header.Set("Authorization", "Bearer "+ownerSession.AccessToken)
	ownerDeleteReq.AddCookie(ownerCookie)
	ownerDeleteRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(ownerDeleteRec, ownerDeleteReq)
	if ownerDeleteRec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, ownerDeleteRec.Code)
	}

	memberDeleteReq := httptest.NewRequest(http.MethodDelete, "/api/account", nil)
	memberDeleteReq.Header.Set("Authorization", "Bearer "+memberSession.AccessToken)
	memberDeleteReq.AddCookie(memberCookie)
	memberDeleteRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(memberDeleteRec, memberDeleteReq)
	if memberDeleteRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, memberDeleteRec.Code, memberDeleteRec.Body.String())
	}
}

func TestOwnerCanDisableUserAndInvalidateSessions(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)
	updateSystemSettings(t, ctx, ownerSession.AccessToken, `{"publicRegistrationEnabled":true}`)

	memberSession, memberCookie := registerUser(t, ctx, "member", "member@example.com", "member1234")
	member := findUserByUsername(t, ctx, "member")

	disableReq := httptest.NewRequest(http.MethodPost, "/api/management/users/"+strconv.FormatInt(member.ID, 10)+"/disable", nil)
	disableReq.Header.Set("Authorization", "Bearer "+ownerSession.AccessToken)
	disableRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(disableRec, disableReq)

	if disableRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, disableRec.Code, disableRec.Body.String())
	}

	assertUnauthorizedMeAndRefresh(t, ctx, memberSession.AccessToken, memberCookie)
}

func TestRegularUserCannotAccessManagementSurfaces(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)
	updateSystemSettings(t, ctx, ownerSession.AccessToken, `{"publicRegistrationEnabled":true}`)
	memberSession, _ := registerUser(t, ctx, "member", "member@example.com", "member1234")

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/system/settings"},
		{method: http.MethodGet, path: "/api/management/users"},
		{method: http.MethodGet, path: "/api/management/system-logs/stream"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+memberSession.AccessToken)
		rec := httptest.NewRecorder()
		ctx.handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status %d for %s %s, got %d", http.StatusForbidden, tc.method, tc.path, rec.Code)
		}
	}
}

func TestManagedUserResponsesEncodeEmptySlicesAsJSONArrays(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)

	listReq := httptest.NewRequest(http.MethodGet, "/api/management/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+ownerSession.AccessToken)
	listRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"actions":[]`) {
		t.Fatalf("expected empty actions to encode as JSON array, got %s", listRec.Body.String())
	}
	if strings.Contains(listRec.Body.String(), "roleKeys") {
		t.Fatalf("expected managed users response to omit roleKeys, got %s", listRec.Body.String())
	}

	createRec := createManagedUser(t, ctx, ownerSession.AccessToken, `{"username":"member","password":"member1234"}`)
	if createRec.Actions == nil || len(createRec.Actions) != 0 {
		t.Fatalf("expected created user actions to encode as empty array, got %+v", createRec.Actions)
	}
}

func TestManagedUserResponsesIncludeLastActiveAt(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	ownerSession, _ := performSetup(t, ctx)

	created := createManagedUser(t, ctx, ownerSession.AccessToken, `{"username":"member","password":"member1234"}`)
	if created.LastActiveAt != nil {
		t.Fatalf("expected newly created managed user to have no last active time, got %v", created.LastActiveAt)
	}

	_, memberCookie := loginUser(t, ctx, "member", "member1234")
	if memberCookie == nil {
		t.Fatal("expected login to create a refresh session")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/management/users?page=1&pageSize=20", nil)
	listReq.Header.Set("Authorization", "Bearer "+ownerSession.AccessToken)
	listRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, listRec.Code, listRec.Body.String())
	}

	var envelope testSuccessEnvelope[managementUsersPageResponse]
	decodeJSONResponse(t, listRec.Body.Bytes(), &envelope)

	for _, item := range envelope.Data.Items {
		if item.ID != created.ID {
			continue
		}
		if item.LastActiveAt == nil {
			t.Fatalf("expected logged-in managed user to have a last active time, got %+v", item)
		}
		return
	}

	t.Fatalf("expected to find managed user %d in list response", created.ID)
}

func TestSPAFallbackAndGzipStillWork(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<title>SPA</title>") {
		t.Fatalf("expected SPA index response, got %q", rec.Body.String())
	}

	gzipReq := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	gzipReq.Header.Set("Accept-Encoding", "gzip")
	gzipRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(gzipRec, gzipReq)
	if gzipRec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip content encoding, got %q", gzipRec.Header().Get("Content-Encoding"))
	}
	if got := gunzipBody(t, gzipRec.Body.Bytes()); !strings.Contains(got, "<title>SPA</title>") {
		t.Fatalf("expected gzipped SPA body, got %q", got)
	}
}

func newTestContext(t *testing.T) *testContext {
	t.Helper()

	dataDir := t.TempDir()
	distDir := createTestDist(t)
	dbRuntime, err := database.Open(context.Background(), database.Config{
		Driver: database.DriverSQLite,
		DSN:    filepath.Join(dataDir, "test.db"),
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := dbRuntime.Close(); err != nil {
			t.Fatalf("failed to close test database: %v", err)
		}
	})

	logStream := logging.NewStream(logging.StreamOptions{
		MaxBytes:  logging.DefaultStreamMaxBytes,
		Retention: 3650 * 24 * time.Hour,
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	authzService := authorization.NewService()
	identityService := identity.NewService(dbRuntime.Identity)
	policiesService := accountpolicies.NewService(dbRuntime.AccountPolicies)
	setupService := setup.NewService(dbRuntime.Setup)
	authService := auth.NewService(auth.Options{
		Secret:             []byte("test-secret"),
		TTL:                time.Hour,
		RefreshIdleTTL:     7 * 24 * time.Hour,
		RefreshAbsoluteTTL: 30 * 24 * time.Hour,
	}, dbRuntime.Sessions, identityService, authzService, policiesService)

	handler := NewHandlerWithOptions(HandlerOptions{
		Logger:          logger,
		LogStream:       logStream,
		Auth:            authService,
		Authorization:   authzService,
		Identity:        identityService,
		Setup:           setupService,
		AccountPolicies: policiesService,
		DataDir:         dataDir,
		FrontendFS:      os.DirFS(distDir),
		LogAPIRequests:  false,
	})

	return &testContext{
		handler:         handler,
		authService:     authService,
		identityService: identityService,
		policiesService: policiesService,
		setupService:    setupService,
		logStream:       logStream,
		dataDir:         dataDir,
	}
}

func (ctx *testContext) authServiceCookieName() string {
	return auth.ResolveRefreshCookieName(auth.Options{Secret: []byte("test-secret")})
}

func performSetup(t *testing.T, ctx *testContext) (sessionResponse, *http.Cookie) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/install/setup", strings.NewReader(`{"username":"owner","password":"owner1234"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var envelope testSuccessEnvelope[sessionResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &envelope)
	return envelope.Data, findCookie(rec.Result().Cookies(), ctx.authServiceCookieName())
}

func updateSystemSettings(t *testing.T, ctx *testContext, token string, payload string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPatch, "/api/system/settings", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func registerUser(t *testing.T, ctx *testContext, username string, email string, password string) (sessionResponse, *http.Cookie) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"username":"`+username+`","email":"`+email+`","password":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var envelope testSuccessEnvelope[sessionResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &envelope)
	return envelope.Data, findCookie(rec.Result().Cookies(), ctx.authServiceCookieName())
}

func loginUser(t *testing.T, ctx *testContext, identifier string, password string) (sessionResponse, *http.Cookie) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"identifier":"`+identifier+`","password":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var envelope testSuccessEnvelope[sessionResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &envelope)
	return envelope.Data, findCookie(rec.Result().Cookies(), ctx.authServiceCookieName())
}

func createManagedUser(t *testing.T, ctx *testContext, token string, payload string) managedUserResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/management/users", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var envelope testSuccessEnvelope[managedUserResponse]
	decodeJSONResponse(t, rec.Body.Bytes(), &envelope)
	return envelope.Data
}

func findUserByUsername(t *testing.T, ctx *testContext, username string) identity.User {
	t.Helper()

	list, err := ctx.identityService.ListUsers(context.Background(), identity.ListUsersParams{Query: username, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	for _, item := range list.Items {
		if item.Username == username {
			return item
		}
	}
	t.Fatalf("user %q not found", username)
	return identity.User{}
}

func assertUnauthorizedMeAndRefresh(t *testing.T, ctx *testContext, accessToken string, refreshCookie *http.Cookie) {
	t.Helper()

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+accessToken)
	meRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, meRec.Code)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshReq.AddCookie(refreshCookie)
	refreshRec := httptest.NewRecorder()
	ctx.handler.ServeHTTP(refreshRec, refreshReq)
	if refreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, refreshRec.Code)
	}
}

func createTestDist(t *testing.T) string {
	t.Helper()

	distDir := t.TempDir()
	assetsDir := filepath.Join(distDir, "assets")

	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("failed to create asset directory: %v", err)
	}

	writeTestFile(t, filepath.Join(distDir, "index.html"), "<!doctype html><html><head><title>SPA</title></head><body>index</body></html>")
	writeGzipFile(t, filepath.Join(distDir, "index.html.gz"), "<!doctype html><html><head><title>SPA</title></head><body>index</body></html>")
	writeTestFile(t, filepath.Join(assetsDir, "app.js"), "console.log('spa asset')\n")
	writeGzipFile(t, filepath.Join(assetsDir, "app.js.gz"), "console.log('spa asset')\n")

	return distDir
}

func writeTestFile(t *testing.T, filename string, content string) {
	t.Helper()
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", filename, err)
	}
}

func writeGzipFile(t *testing.T, filename string, content string) {
	t.Helper()

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte(content)); err != nil {
		t.Fatalf("failed to gzip %s: %v", filename, err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to finalize gzip %s: %v", filename, err)
	}
	if err := os.WriteFile(filename, compressed.Bytes(), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", filename, err)
	}
}

func decodeJSONResponse(t *testing.T, payload []byte, dst any) {
	t.Helper()
	if err := json.Unmarshal(payload, dst); err != nil {
		t.Fatalf("failed to decode JSON response: %v; payload=%s", err, string(payload))
	}
}

func decodeErrorCode(t *testing.T, payload []byte, wantCode string) {
	t.Helper()

	var envelope testErrorEnvelope
	decodeJSONResponse(t, payload, &envelope)
	if envelope.Error.Code != wantCode {
		t.Fatalf("expected error code %q, got %q", wantCode, envelope.Error.Code)
	}
}

func gunzipBody(t *testing.T, body []byte) string {
	t.Helper()

	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer reader.Close()

	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to decode gzip body: %v", err)
	}
	return string(decoded)
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
