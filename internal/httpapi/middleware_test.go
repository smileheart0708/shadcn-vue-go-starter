package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/identity"
)

type missingBearerAuthService struct{}

func (missingBearerAuthService) AuthenticateRequest(context.Context, string) (auth.Actor, error) {
	return auth.Actor{}, errors.Join(errors.New("wrapped"), auth.ErrMissingBearerToken)
}

func TestRequireAuthRecognizesWrappedMissingBearerToken(t *testing.T) {
	t.Parallel()

	handler := RequireAuth(missingBearerAuthService{})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	decodeErrorCode(t, rec.Body.Bytes(), string(CodeUnauthorized))
}

// A database outage must not look like an expired session: answering 401 makes the
// client throw away a valid session and hides the cause. Only errors that classify the
// token itself may answer 401; everything else stays a 500.
func TestRequireAuthMapsInfrastructureErrorsToInternalError(t *testing.T) {
	t.Parallel()

	handler := RequireAuth(failingAuthenticator{
		err: errors.Join(errors.New("auth: resolve actor"), errors.New("dial tcp: connection refused")),
	})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/me", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	decodeErrorCode(t, rec.Body.Bytes(), string(CodeInternalError))
}

func TestRequireAuthMapsInvalidTokenToUnauthorized(t *testing.T) {
	t.Parallel()

	handler := RequireAuth(failingAuthenticator{
		err: errors.Join(errors.New("wrapped"), auth.ErrInvalidToken),
	})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/me", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	decodeErrorCode(t, rec.Body.Bytes(), string(CodeInvalidToken))
}

func TestRequireAuthAcceptsAuthenticatedActor(t *testing.T) {
	t.Parallel()

	authenticator := staticAuthenticator{actor: auth.Actor{
		Principal: auth.Principal{UserID: 1, SessionID: "session", SecurityVersion: 1},
		User:      identity.User{ID: 1, Username: "owner", Role: identity.RoleOwner, Status: identity.StatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	handler := RequireAuth(authenticator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.ActorFromContext(r.Context()); !ok {
			t.Fatal("expected actor in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestChainAppliesMiddlewaresInDeclaredOrder(t *testing.T) {
	t.Parallel()

	var calls []string
	middleware := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, name+":before")
				next.ServeHTTP(w, r)
				calls = append(calls, name+":after")
			})
		}
	}

	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls = append(calls, "handler")
			w.WriteHeader(http.StatusNoContent)
		}),
		middleware("first"),
		middleware("second"),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	want := []string{"first:before", "second:before", "handler", "second:after", "first:after"}
	if len(calls) != len(want) {
		t.Fatalf("expected calls %v, got %v", want, calls)
	}
	for i, call := range calls {
		if call != want[i] {
			t.Fatalf("expected calls %v, got %v", want, calls)
		}
	}
}

func TestRequireCapabilityGatesOnGrantedCapabilities(t *testing.T) {
	t.Parallel()

	actor := auth.Actor{
		UserID: 7, SessionID: "session", SecurityVersion: 1,
		User: identity.User{ID: 7, Username: "member", Role: identity.RoleUser, Status: identity.StatusActive},
		Authorization: authorization.ViewerAuthorization{
			Role:         identity.RoleUser,
			Capabilities: []authorization.Capability{authorization.CapabilityAccountDeleteSelf},
		},
	}

	var reached bool
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			reached = true
			w.WriteHeader(http.StatusNoContent)
		}),
		RequireAuth(staticAuthenticator{actor: actor}),
		RequireCapability(authorization.CapabilityManagementUsersRead),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/management/users", nil))

	if reached {
		t.Fatal("expected the guarded handler to be skipped")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	decodeErrorCode(t, rec.Body.Bytes(), string(CodeForbidden))
	if strings.Contains(rec.Body.String(), string(authorization.CapabilityManagementUsersRead)) {
		t.Fatalf("expected the response to hide the required capability, got %s", rec.Body.String())
	}

	granted := actor
	granted.Authorization.Capabilities = append(granted.Authorization.Capabilities, authorization.CapabilityManagementUsersRead)
	allowedHandler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		RequireAuth(staticAuthenticator{actor: granted}),
		RequireCapability(authorization.CapabilityManagementUsersRead),
	)

	allowedRec := httptest.NewRecorder()
	allowedHandler.ServeHTTP(allowedRec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/management/users", nil))
	if allowedRec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, allowedRec.Code)
	}
}

type failingAuthenticator struct {
	err error
}

func (f failingAuthenticator) AuthenticateRequest(context.Context, string) (auth.Actor, error) {
	return auth.Actor{}, f.err
}

type staticAuthenticator struct {
	actor auth.Actor
}

func (s staticAuthenticator) AuthenticateRequest(context.Context, string) (auth.Actor, error) {
	return s.actor, nil
}
