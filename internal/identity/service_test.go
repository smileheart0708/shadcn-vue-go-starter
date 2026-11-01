package identity

import (
	"context"
	"errors"
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
