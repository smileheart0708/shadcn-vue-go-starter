package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/identity"
	"main/internal/setup"
)

func TestOpenConfiguresSQLitePragmasAndSingleConnectionPool(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	db := store.db

	var foreignKeys int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	var busyTimeout int64
	if err := db.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if busyTimeout < defaultBusyTimeout.Milliseconds() {
		t.Fatalf("busy_timeout = %dms, want at least %dms", busyTimeout, defaultBusyTimeout.Milliseconds())
	}

	var journalMode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}
	if stats := db.Stats(); stats.MaxOpenConnections != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", stats.MaxOpenConnections)
	}
}

func TestMigrateRejectsModifiedAppliedMigration(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	if _, err := store.db.Exec(`UPDATE schema_migrations SET checksum = 'tampered' WHERE version = 1`); err != nil {
		t.Fatalf("tamper migration checksum: %v", err)
	}
	if err := migrate(context.Background(), store.db); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("migrate() error = %v, want checksum mismatch", err)
	}
}

func TestMigrateAllowsConcurrentStartup(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "app.db")
	db1, close1 := openRawSQLiteDB(t, path)
	defer close1()
	db2, close2 := openRawSQLiteDB(t, path)
	defer close2()

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, db := range []*sql.DB{db1, db2} {
		wg.Go(func() {
			<-start
			errs <- migrate(context.Background(), db)
		})
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent migrate() error = %v", err)
		}
	}

	var count int
	if err := db1.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count applied migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("applied migration count = %d, want 1", count)
	}
}

func TestStoreRepositoriesKeepDomainWritesAtomic(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
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
		t.Fatal("expected disable to revoke active sessions")
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
		t.Fatalf("password update result = %+v, want new hash and security version increment", after)
	}
	passwordSession, err := store.GetSession(ctx, "password-session")
	if err != nil {
		t.Fatalf("get password session: %v", err)
	}
	if passwordSession.RevokedAt == nil {
		t.Fatal("expected password change to revoke active sessions")
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return store
}

func openRawSQLiteDB(t *testing.T, path string) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatalf("open raw SQLite DB: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		t.Fatalf("ping raw SQLite DB: %v", err)
	}
	return db, func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close raw SQLite DB: %v", err)
		}
	}
}
