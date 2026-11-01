package database

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/identity"
	"main/internal/setup"
)

func TestOpenSQLiteProvidesDomainRepositories(t *testing.T) {
	t.Parallel()

	runtime := openTestRuntime(t)

	if runtime.Driver() != DriverSQLite {
		t.Fatalf("Driver() = %q, want %q", runtime.Driver(), DriverSQLite)
	}
}

func TestOpenRejectsDriverNotCompiledIntoBinary(t *testing.T) {
	t.Parallel()

	const dsn = "postgres://user:secret@example.test/app"
	_, err := Open(context.Background(), Config{
		Driver: "postgres",
		DSN:    dsn,
	})
	if err == nil {
		t.Fatal("expected unsupported driver error")
	}
	if strings.Contains(err.Error(), dsn) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unsupported driver error leaks the DSN: %v", err)
	}
}

func TestRuntimeKeepsDomainWritesAtomic(t *testing.T) {
	t.Parallel()

	runtime := openTestRuntime(t)
	store := runtime.Store
	ctx := context.Background()
	setupService := setup.NewService(store)
	policiesService := accountpolicies.NewService(store)
	identityService := identity.NewService(store)

	owner, err := setupService.Complete(ctx, setup.CompleteSetupInput{
		Username: "owner",
		Password: "owner1234",
	})
	if err != nil {
		t.Fatalf("complete setup: %v", err)
	}
	state, err := setupService.GetState(ctx)
	if err != nil {
		t.Fatalf("get setup state: %v", err)
	}
	if !state.SetupCompleted || state.OwnerUserID == nil || *state.OwnerUserID != owner.ID {
		t.Fatalf("unexpected completed state: %+v", state)
	}
	if _, err := setupService.Complete(ctx, setup.CompleteSetupInput{Username: "other", Password: "other1234"}); !errors.Is(err, setup.ErrAlreadyCompleted) {
		t.Fatalf("second setup error = %v, want ErrAlreadyCompleted", err)
	}

	enabled := true
	if _, err := policiesService.Update(ctx, accountpolicies.UpdateInput{PublicRegistrationEnabled: &enabled}); err != nil {
		t.Fatalf("enable registration: %v", err)
	}
	if _, err := policiesService.Update(ctx, accountpolicies.UpdateInput{SelfServiceAccountDeletionEnabled: &enabled}); err != nil {
		t.Fatalf("enable self-service deletion: %v", err)
	}
	policies, err := policiesService.Get(ctx)
	if err != nil {
		t.Fatalf("get policies: %v", err)
	}
	if !policies.PublicRegistrationEnabled || !policies.SelfServiceAccountDeletionEnabled {
		t.Fatalf("partial policy updates lost a field: %+v", policies)
	}

	user, err := identityService.CreateUser(ctx, identity.CreateUserParams{
		Username:     "member",
		PasswordHash: "hash",
		Role:         identity.RoleUser,
	})
	if err != nil {
		t.Fatalf("create member: %v", err)
	}
	if _, err := identityService.CreateUser(ctx, identity.CreateUserParams{
		Username:     "member",
		PasswordHash: "hash",
		Role:         identity.RoleUser,
	}); !errors.Is(err, identity.ErrUsernameTaken) {
		t.Fatalf("duplicate user error = %v, want ErrUsernameTaken", err)
	}

	now := time.Now().UTC()
	if err := store.CreateSession(ctx, auth.Session{
		ID:               "disable-session",
		UserID:           user.ID,
		RefreshTokenHash: "hash",
		CreatedAt:        now,
		LastUsedAt:       now,
		ExpiresAt:        now.Add(time.Hour),
		IdleExpiresAt:    now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("create disable session: %v", err)
	}
	if _, err := identityService.SetUserStatus(ctx, user.ID, identity.StatusDisabled); err != nil {
		t.Fatalf("disable user: %v", err)
	}
	disabledSession, err := store.GetSession(ctx, "disable-session")
	if err != nil {
		t.Fatalf("get disabled session: %v", err)
	}
	if disabledSession.RevokedAt == nil {
		t.Fatal("expected disabling the user to revoke active sessions")
	}

	passwordUser, err := identityService.CreateUser(ctx, identity.CreateUserParams{
		Username:     "password-user",
		PasswordHash: "old-hash",
		Role:         identity.RoleUser,
	})
	if err != nil {
		t.Fatalf("create password user: %v", err)
	}
	if err := store.CreateSession(ctx, auth.Session{
		ID:               "password-session",
		UserID:           passwordUser.ID,
		RefreshTokenHash: "hash",
		CreatedAt:        now,
		LastUsedAt:       now,
		ExpiresAt:        now.Add(time.Hour),
		IdleExpiresAt:    now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("create password session: %v", err)
	}
	before, err := identityService.GetAuthRecordByID(ctx, passwordUser.ID)
	if err != nil {
		t.Fatalf("get password user before update: %v", err)
	}
	if err := store.UpdatePasswordAndRevokeSessions(ctx, passwordUser.ID, "new-hash", now); err != nil {
		t.Fatalf("update password and revoke sessions: %v", err)
	}
	after, err := identityService.GetAuthRecordByID(ctx, passwordUser.ID)
	if err != nil {
		t.Fatalf("get password user after update: %v", err)
	}
	if after.PasswordHash != "new-hash" || after.SecurityVersion != before.SecurityVersion+1 {
		t.Fatalf("password update result = %+v, want the new hash and an incremented security version", after)
	}
	passwordSession, err := store.GetSession(ctx, "password-session")
	if err != nil {
		t.Fatalf("get password session: %v", err)
	}
	if passwordSession.RevokedAt == nil {
		t.Fatal("expected changing the password to revoke active sessions")
	}
}

func openTestRuntime(t *testing.T) *Runtime {
	t.Helper()

	runtime, err := Open(context.Background(), Config{
		Driver: DriverSQLite,
		DSN:    filepath.Join(t.TempDir(), "app.db"),
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := runtime.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})
	return runtime
}
