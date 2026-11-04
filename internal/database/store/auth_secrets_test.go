package store

import (
	"context"
	"errors"
	"testing"

	"main/internal/auth"
)

func TestJWTSecretRoundTrip(t *testing.T) {
	t.Parallel()

	repository := newTestStore(t)
	ctx := context.Background()

	if _, err := repository.GetJWTSecret(ctx); !errors.Is(err, auth.ErrJWTSecretNotFound) {
		t.Fatalf("GetJWTSecret() error = %v, want ErrJWTSecretNotFound", err)
	}

	const secret = "0123456789abcdef0123456789abcdef"
	effective, err := repository.EnsureJWTSecret(ctx, secret)
	if err != nil {
		t.Fatalf("EnsureJWTSecret() error = %v", err)
	}
	if effective != secret {
		t.Fatalf("EnsureJWTSecret() = %q, want %q", effective, secret)
	}

	const competing = "fedcba9876543210fedcba9876543210"
	effective, err = repository.EnsureJWTSecret(ctx, competing)
	if err != nil {
		t.Fatalf("second EnsureJWTSecret() error = %v", err)
	}
	if effective != secret {
		t.Fatalf("second EnsureJWTSecret() = %q, want the first stored secret %q", effective, secret)
	}

	stored, err := repository.GetJWTSecret(ctx)
	if err != nil {
		t.Fatalf("GetJWTSecret() error = %v", err)
	}
	if stored != secret {
		t.Fatalf("GetJWTSecret() = %q, want %q", stored, secret)
	}
}
