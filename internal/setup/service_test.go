package setup_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"main/internal/database"
	"main/internal/identity"
	"main/internal/setup"
)

func TestCompleteOnlyRunsOnceAndPersistsOwner(t *testing.T) {
	t.Parallel()

	runtime, err := database.Open(context.Background(), database.Config{
		Driver: database.DriverSQLite,
		DSN:    filepath.Join(t.TempDir(), "app.db"),
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	})

	service := setup.NewService(runtime.Setup)

	owner, err := service.Complete(context.Background(), setup.CompleteSetupInput{
		Username: "owner",
		Password: "owner1234",
	})
	if err != nil {
		t.Fatalf("failed to complete setup: %v", err)
	}
	if owner.Email != nil {
		t.Fatalf("expected setup owner email to be nil, got %q", *owner.Email)
	}
	if owner.Role != "owner" {
		t.Fatalf("expected owner role, got %+v", owner.Role)
	}

	_, err = service.Complete(context.Background(), setup.CompleteSetupInput{
		Username: "other",
		Password: "password123",
	})
	if err != setup.ErrAlreadyCompleted {
		t.Fatalf("expected ErrAlreadyCompleted, got %v", err)
	}

	state, err := service.GetState(context.Background())
	if err != nil {
		t.Fatalf("failed to read setup state: %v", err)
	}
	if !state.SetupCompleted {
		t.Fatal("expected setup state to be completed")
	}
	if state.OwnerUserID == nil || *state.OwnerUserID != owner.ID {
		t.Fatalf("expected owner ID %d in setup state, got %v", owner.ID, state.OwnerUserID)
	}
}

func TestCompleteRejectsBlankUsername(t *testing.T) {
	t.Parallel()

	// The check must run before storage is touched, so an embedded nil repository — which
	// panics on any call — is enough and proves that ordering.
	service := setup.NewService(struct{ setup.Repository }{})

	_, err := service.Complete(context.Background(), setup.CompleteSetupInput{
		Username: "   ",
		Password: "password123",
	})
	if !errors.Is(err, identity.ErrUsernameRequired) {
		t.Fatalf("expected identity.ErrUsernameRequired, got %v", err)
	}
}
