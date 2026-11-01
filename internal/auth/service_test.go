package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"main/internal/identity"
)

// The refresh handler decides between 401 (drop the cookie) and 500 (keep it so a later
// page load can recover) with this predicate. Missing an error here means a real failure
// is reported as an expired session, so every session error is listed explicitly.
func TestIsSessionInvalidError(t *testing.T) {
	t.Parallel()

	sessionErrors := []error{
		ErrInvalidRefreshToken,
		ErrTokenReuseDetected,
		ErrRefreshTokenRevoked,
		ErrRefreshTokenExpired,
		ErrAccountDisabled,
	}
	for _, err := range sessionErrors {
		if !IsSessionInvalidError(err) {
			t.Fatalf("expected %v to be classified as a session error", err)
		}
	}

	infrastructureErrors := []error{
		ErrInvalidCredentials,
		ErrSessionNotFound,
		errors.New("database is down"),
	}
	for _, err := range infrastructureErrors {
		if IsSessionInvalidError(err) {
			t.Fatalf("expected %v to be classified as an infrastructure failure", err)
		}
	}

	if IsSessionInvalidError(nil) {
		t.Fatal("expected nil not to be classified as a session error")
	}
}

// CreateManagedUser owns password validation for the administrative path. The zero-value
// service is enough: the length check runs before any repository access.
func TestCreateManagedUserRejectsShortPassword(t *testing.T) {
	t.Parallel()

	service := &Service{}
	_, err := service.CreateManagedUser(context.Background(), CreateManagedUserParams{
		Username: "alice",
		Password: strings.Repeat("a", MinPasswordLength-1),
	})

	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

// The handler used to check the username first; validation moved into the service must
// keep that order so two invalid fields still answer username_required, and the expensive
// Argon2 hash is not paid for an invalid username.
func TestCreateManagedUserValidatesUsernameBeforePassword(t *testing.T) {
	t.Parallel()

	service := &Service{}
	_, err := service.CreateManagedUser(context.Background(), CreateManagedUserParams{
		Username: "   ",
		Password: strings.Repeat("a", MinPasswordLength-1),
	})

	if !errors.Is(err, identity.ErrUsernameRequired) {
		t.Fatalf("expected identity.ErrUsernameRequired, got %v", err)
	}
}

// managedUserRepositoryStub only implements CreateUser; the embedded interface makes any
// other repository call panic, which is what a test wants.
type managedUserRepositoryStub struct {
	identity.Repository
	params identity.CreateUserParams
}

func (s *managedUserRepositoryStub) CreateUser(_ context.Context, params identity.CreateUserParams) (identity.User, error) {
	s.params = params
	return identity.User{
		ID:       42,
		Username: params.Username,
		Email:    params.Email,
		Role:     params.Role,
		Status:   identity.StatusActive,
	}, nil
}

// Administrative creation always produces a plain user and stores only the hash. Both are
// pinned so a future change cannot start accepting a role from the request or persist the
// plaintext password.
func TestCreateManagedUserCreatesPlainUserWithHashedPassword(t *testing.T) {
	t.Parallel()

	repository := &managedUserRepositoryStub{}
	service := &Service{identities: identity.NewService(repository)}
	email := "  Alice@Example.COM "
	password := "correct horse battery staple"

	created, err := service.CreateManagedUser(context.Background(), CreateManagedUserParams{
		Username: "  alice  ",
		Email:    &email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("CreateManagedUser() error = %v", err)
	}

	if repository.params.Role != identity.RoleUser {
		t.Fatalf("role = %q, want %q", repository.params.Role, identity.RoleUser)
	}
	if repository.params.Username != "alice" {
		t.Fatalf("username = %q, want the normalized value", repository.params.Username)
	}
	if repository.params.Email == nil || *repository.params.Email != "alice@example.com" {
		t.Fatalf("email = %v, want the normalized value", repository.params.Email)
	}
	if repository.params.PasswordHash == "" || repository.params.PasswordHash == password {
		t.Fatal("expected a password hash, not the plaintext password")
	}
	matches, err := VerifyPassword(password, repository.params.PasswordHash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !matches {
		t.Fatal("expected the stored hash to verify the original password")
	}
	if created.ID != 42 || created.Role != identity.RoleUser {
		t.Fatalf("created user = %+v, want the repository result", created)
	}
}
