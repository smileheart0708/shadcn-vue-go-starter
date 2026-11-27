package identity

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// createUserStub records what reached the repository. Only CreateUser is implemented;
// the embedded interface makes any other call panic, which is what a test wants.
type createUserStub struct {
	Repository

	called bool
	params CreateUserParams
}

func (s *createUserStub) CreateUser(_ context.Context, params CreateUserParams) (User, error) {
	s.called = true
	s.params = params
	return User{Username: params.Username, Role: params.Role}, nil
}

// Validation must stop before the repository: ErrUsernameRequired is mapped by handlers,
// while the role/hash checks guard internal invariants and must never reach storage.
func TestCreateUserReportsValidationSentinels(t *testing.T) {
	t.Parallel()

	invalidEmail := "a@b"
	tests := []struct {
		name   string
		params CreateUserParams
		want   error
	}{
		{
			name:   "blank username",
			params: CreateUserParams{Username: "   ", Role: RoleUser, PasswordHash: "hash"},
			want:   ErrUsernameRequired,
		},
		{
			name:   "username that looks like an email",
			params: CreateUserParams{Username: "carol@example.com", Role: RoleUser, PasswordHash: "hash"},
			want:   ErrUsernameInvalid,
		},
		{
			name:   "invalid email",
			params: CreateUserParams{Username: "alice", Email: &invalidEmail, Role: RoleUser, PasswordHash: "hash"},
			want:   ErrEmailInvalid,
		},
		{
			name:   "unknown role",
			params: CreateUserParams{Username: "alice", Role: "auditor", PasswordHash: "hash"},
			want:   errRoleRequired,
		},
		{
			name:   "blank password hash",
			params: CreateUserParams{Username: "alice", Role: RoleUser, PasswordHash: "  "},
			want:   errPasswordHashRequired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &createUserStub{}
			service := NewService(repository)

			_, err := service.CreateUser(context.Background(), test.params)
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
			if repository.called {
				t.Fatal("expected validation to stop before the repository")
			}
		})
	}
}

type updateProfileStub struct {
	Repository

	called   bool
	username string
}

func (s *updateProfileStub) UpdateProfile(_ context.Context, _ int64, username string, _ *string) (User, error) {
	s.called = true
	s.username = username
	return User{ID: 1, Username: username}, nil
}

// UpdateProfile owns the only write path to the username column, so it must reject a
// blank one itself: handlers no longer pre-check it, and UpdateManagedUser relies on
// this method instead of repeating the rule.
func TestUpdateProfileRejectsBlankUsernameBeforeRepository(t *testing.T) {
	t.Parallel()

	repository := &updateProfileStub{}
	service := NewService(repository)

	if _, err := service.UpdateProfile(context.Background(), 1, "   ", nil); !errors.Is(err, ErrUsernameRequired) {
		t.Fatalf("UpdateProfile(blank) error = %v, want ErrUsernameRequired", err)
	}
	if _, err := service.UpdateManagedUser(context.Background(), 1, UpdateManagedUserParams{Username: "\t"}); !errors.Is(err, ErrUsernameRequired) {
		t.Fatalf("UpdateManagedUser(blank) error = %v, want ErrUsernameRequired", err)
	}
	if _, err := service.UpdateManagedUser(context.Background(), 1, UpdateManagedUserParams{Username: "carol@example.com"}); !errors.Is(err, ErrUsernameInvalid) {
		t.Fatalf("UpdateManagedUser(email-shaped username) error = %v, want ErrUsernameInvalid", err)
	}
	invalidEmail := "a@b"
	if _, err := service.UpdateProfile(context.Background(), 1, "alice", &invalidEmail); !errors.Is(err, ErrEmailInvalid) {
		t.Fatalf("UpdateProfile(invalid email) error = %v, want ErrEmailInvalid", err)
	}
	if repository.called {
		t.Fatal("expected validation to stop before the repository")
	}
}

func TestCreateUserPassesNormalizedParamsToRepository(t *testing.T) {
	t.Parallel()

	repository := &createUserStub{}
	service := NewService(repository)
	email := "  Alice@Example.COM "

	if _, err := service.CreateUser(context.Background(), CreateUserParams{
		Username:     "  alice  ",
		Email:        &email,
		Role:         RoleUser,
		PasswordHash: "hash",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repository.params.Username != "alice" {
		t.Fatalf("expected trimmed username, got %q", repository.params.Username)
	}
	if repository.params.Email == nil || *repository.params.Email != "alice@example.com" {
		t.Fatalf("expected lowercase email, got %v", repository.params.Email)
	}
}

func TestValidateUsername(t *testing.T) {
	t.Parallel()

	valid := []string{
		"a",
		"ab",
		"alice",
		"Alice-01",
		"a.b_c-d",
		"a..b",
		strings.Repeat("a", 32),
	}
	for _, username := range valid {
		if err := ValidateUsername(username); err != nil {
			t.Errorf("expected %q to be accepted, got %v", username, err)
		}
	}

	invalid := []string{
		"alice@example.com",
		"alice bob",
		"alice!",
		"alice/",
		".alice",
		"alice.",
		"-alice",
		"alice-",
		"_alice",
		"alice_",
		"中文",
		"😀",
		"ali\nce",
		"ad\u200Bmin",
		"admin\u202Egnp.exe",
		strings.Repeat("a", 33),
	}
	for _, username := range invalid {
		if err := ValidateUsername(username); !errors.Is(err, ErrUsernameInvalid) {
			t.Errorf("expected %q to be rejected, got %v", username, err)
		}
	}

	if err := ValidateUsername("   "); !errors.Is(err, ErrUsernameRequired) {
		t.Fatalf("blank username error = %v, want ErrUsernameRequired", err)
	}
}

// The Go pattern is a port of zod's z.email(), which the frontend uses on form input.
// These cases pin the differences that matter: a bare domain is not enough, the local
// part may not contain consecutive dots, and the TLD is at least two letters.
func TestEmailPatternMatchesFrontendValidation(t *testing.T) {
	t.Parallel()

	valid := []string{
		"alice@example.com",
		"a.b+tag@example.co.uk",
		"o'hara@example.com",
		"alice@example-.com", // zod accepts a trailing hyphen in a label
		"a@b.co",
	}
	for _, email := range valid {
		if !emailPattern.MatchString(email) {
			t.Errorf("expected %q to be accepted", email)
		}
	}

	invalid := []string{
		"a@b",
		"alice@example",
		"alice..b@example.com",
		"alice@example..com",
		".alice@example.com",
		"alice.@example.com",
		"alice@example.c",
		"alice@example.c0m",
		"alice@example.中国",
	}
	for _, email := range invalid {
		if emailPattern.MatchString(email) {
			t.Errorf("expected %q to be rejected", email)
		}
	}
}
