package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/uptrace/bun"

	"main/internal/auth"
	"main/internal/database/dialect/sqlite"
	"main/internal/database/migrate"
	"main/internal/identity"
)

func TestCreateUserReturnsGeneratedIDAndDomainErrors(t *testing.T) {
	t.Parallel()

	repository := newTestStore(t)
	ctx := context.Background()

	email := "owner@example.test"
	created, err := repository.CreateUser(ctx, identity.CreateUserParams{
		Username:     "owner",
		Email:        &email,
		PasswordHash: "hash",
		Role:         identity.RoleOwner,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if created.ID <= 0 {
		t.Fatalf("created user ID = %d, want a generated id", created.ID)
	}
	if created.LastActiveAt != nil {
		t.Fatalf("LastActiveAt = %v, want nil before any session", created.LastActiveAt)
	}

	if _, err := repository.CreateUser(ctx, identity.CreateUserParams{
		Username:     "owner",
		PasswordHash: "hash",
		Role:         identity.RoleUser,
	}); !errors.Is(err, identity.ErrUsernameTaken) {
		t.Fatalf("duplicate username error = %v, want ErrUsernameTaken", err)
	}

	if _, err := repository.CreateUser(ctx, identity.CreateUserParams{
		Username:     "someone",
		Email:        &email,
		PasswordHash: "hash",
		Role:         identity.RoleUser,
	}); !errors.Is(err, identity.ErrEmailTaken) {
		t.Fatalf("duplicate email error = %v, want ErrEmailTaken", err)
	}

	if _, err := repository.CreateUser(ctx, identity.CreateUserParams{
		Username:     "second-owner",
		PasswordHash: "hash",
		Role:         identity.RoleOwner,
	}); !errors.Is(err, identity.ErrOwnerAlreadyExists) {
		t.Fatalf("second owner error = %v, want ErrOwnerAlreadyExists", err)
	}

	record, err := repository.GetAuthRecordByIdentifier(ctx, "OWNER")
	if err != nil {
		t.Fatalf("GetAuthRecordByIdentifier() error = %v", err)
	}
	if record.ID != created.ID || record.PasswordHash != "hash" {
		t.Fatalf("auth record = %+v, want id %d and the stored hash", record, created.ID)
	}
}

func TestListUsersFiltersPagesAndReportsLastActivity(t *testing.T) {
	t.Parallel()

	repository := newTestStore(t)
	ctx := context.Background()

	created := make([]identity.User, 0, 3)
	for _, username := range []string{"alice", "bob", "carol"} {
		user, err := repository.CreateUser(ctx, identity.CreateUserParams{
			Username:     username,
			PasswordHash: "hash",
			Role:         identity.RoleUser,
		})
		if err != nil {
			t.Fatalf("CreateUser(%q) error = %v", username, err)
		}
		created = append(created, user)
	}

	lastUsedAt := time.Now().UTC().Add(-time.Minute)
	if err := repository.CreateSession(ctx, auth.Session{
		ID:               "alice-session",
		UserID:           created[0].ID,
		RefreshTokenHash: "hash",
		CreatedAt:        lastUsedAt,
		LastUsedAt:       lastUsedAt,
		ExpiresAt:        lastUsedAt.Add(time.Hour),
		IdleExpiresAt:    lastUsedAt.Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	page, err := repository.ListUsers(ctx, identity.ListUsersParams{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if page.Total != 3 || len(page.Items) != 2 {
		t.Fatalf("ListUsers() = total %d, items %d; want total 3, items 2", page.Total, len(page.Items))
	}

	secondPage, err := repository.ListUsers(ctx, identity.ListUsersParams{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("ListUsers(page 2) error = %v", err)
	}
	if len(secondPage.Items) != 1 {
		t.Fatalf("second page items = %d, want 1", len(secondPage.Items))
	}

	filtered, err := repository.ListUsers(ctx, identity.ListUsersParams{Query: "ALI", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListUsers(query) error = %v", err)
	}
	if filtered.Total != 1 || len(filtered.Items) != 1 || filtered.Items[0].Username != "alice" {
		t.Fatalf("case-insensitive query returned %+v, want only alice", filtered.Items)
	}
	if filtered.Items[0].LastActiveAt == nil {
		t.Fatal("LastActiveAt = nil, want the session activity timestamp")
	}

	if _, err := repository.SetUserStatus(ctx, created[1].ID, identity.StatusDisabled); err != nil {
		t.Fatalf("SetUserStatus() error = %v", err)
	}
	disabled, err := repository.ListUsers(ctx, identity.ListUsersParams{Status: identity.StatusDisabled, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListUsers(status) error = %v", err)
	}
	if disabled.Total != 1 || disabled.Items[0].Username != "bob" {
		t.Fatalf("status filter returned %+v, want only bob", disabled.Items)
	}
}

func TestRotateSessionRequiresMatchingHashAndActiveSession(t *testing.T) {
	t.Parallel()

	repository := newTestStore(t)
	ctx := context.Background()

	user, err := repository.CreateUser(ctx, identity.CreateUserParams{
		Username:     "rotator",
		PasswordHash: "hash",
		Role:         identity.RoleUser,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	now := time.Now().UTC()
	if err := repository.CreateSession(ctx, auth.Session{
		ID:               "session",
		UserID:           user.ID,
		RefreshTokenHash: "first-hash",
		CreatedAt:        now,
		LastUsedAt:       now,
		ExpiresAt:        now.Add(time.Hour),
		IdleExpiresAt:    now.Add(time.Hour),
		IP:               ptr("127.0.0.1"),
		UserAgent:        ptr("test-agent"),
	}); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	rotated, err := repository.RotateSession(ctx, "session", "first-hash", "second-hash", now, now.Add(time.Hour))
	if err != nil || !rotated {
		t.Fatalf("RotateSession() = (%v, %v), want (true, nil)", rotated, err)
	}

	replayed, err := repository.RotateSession(ctx, "session", "first-hash", "third-hash", now, now.Add(time.Hour))
	if err != nil || replayed {
		t.Fatalf("RotateSession(replayed) = (%v, %v), want (false, nil)", replayed, err)
	}

	session, err := repository.GetSession(ctx, "session")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if session.RefreshTokenHash != "second-hash" {
		t.Fatalf("stored hash = %q, want second-hash", session.RefreshTokenHash)
	}
	if session.IP == nil || *session.IP != "127.0.0.1" || session.UserAgent == nil || *session.UserAgent != "test-agent" {
		t.Fatalf("session metadata = %+v, want the stored client details", session)
	}

	if err := repository.RevokeSession(ctx, "session", "logout", now); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	revoked, err := repository.RotateSession(ctx, "session", "second-hash", "fourth-hash", now, now.Add(time.Hour))
	if err != nil || revoked {
		t.Fatalf("RotateSession(revoked) = (%v, %v), want (false, nil)", revoked, err)
	}
}

func TestGetSessionReportsNotFound(t *testing.T) {
	t.Parallel()

	repository := newTestStore(t)
	if _, err := repository.GetSession(context.Background(), "missing"); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("GetSession() error = %v, want ErrSessionNotFound", err)
	}
}

// UpdateProfile serves both the self-service and administrative edit paths, so pin the
// stored fields and the not-found contract the handlers rely on. Atomicity itself is a
// property of the transaction and cannot be asserted deterministically from here.
func TestUpdateProfileWritesFieldsAndReportsMissingUser(t *testing.T) {
	t.Parallel()

	repository := newTestStore(t)
	ctx := context.Background()

	created, err := repository.CreateUser(ctx, identity.CreateUserParams{
		Username:     "alice",
		PasswordHash: "hash",
		Role:         identity.RoleUser,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	email := "alice@example.test"
	updated, err := repository.UpdateProfile(ctx, created.ID, "alice-renamed", &email)
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if updated.ID != created.ID || updated.Username != "alice-renamed" {
		t.Fatalf("UpdateProfile() = %+v, want the updated username", updated)
	}
	if updated.Email == nil || *updated.Email != email {
		t.Fatalf("updated email = %v, want %q", updated.Email, email)
	}

	loaded, err := repository.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if loaded.Username != "alice-renamed" || loaded.Email == nil || *loaded.Email != email {
		t.Fatalf("persisted user = %+v, want the updated fields", loaded)
	}

	if _, err := repository.UpdateProfile(ctx, created.ID+1000, "ghost", nil); !errors.Is(err, identity.ErrUserNotFound) {
		t.Fatalf("UpdateProfile(missing) error = %v, want ErrUserNotFound", err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()

	driver := sqlite.New()
	sqldb, err := driver.Open(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	db := bun.NewDB(sqldb, driver.BunDialect())
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close database: %v", closeErr)
		}
	})
	if migrateErr := migrate.Run(context.Background(), db, driver.Migrations()); migrateErr != nil {
		t.Fatalf("apply migrations: %v", migrateErr)
	}
	return New(db, driver)
}

func ptr[T any](value T) *T {
	return &value
}
