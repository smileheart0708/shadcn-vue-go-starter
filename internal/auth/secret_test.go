package auth

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

const testEnvJWTSecret = "0123456789abcdef0123456789abcdef"

type fakeJWTSecretRepository struct {
	secret     string
	found      bool
	getErr     error
	saveErr    error
	saveResult string
	saved      []string
}

func (r *fakeJWTSecretRepository) GetJWTSecret(context.Context) (string, error) {
	if r.getErr != nil {
		return "", r.getErr
	}
	if !r.found {
		return "", ErrJWTSecretNotFound
	}
	return r.secret, nil
}

func (r *fakeJWTSecretRepository) EnsureJWTSecret(_ context.Context, candidate string) (string, error) {
	if r.saveErr != nil {
		return "", r.saveErr
	}
	if r.saveResult != "" {
		r.secret = r.saveResult
		r.found = true
		return r.saveResult, nil
	}
	r.secret = candidate
	r.found = true
	r.saved = append(r.saved, candidate)
	return candidate, nil
}

func TestResolveJWTSecretPrefersStoredSecret(t *testing.T) {
	t.Parallel()

	repository := &fakeJWTSecretRepository{secret: "stored-secret", found: true}

	// A weak seed is ignored too: it must not fail a start that already has one.
	secret, err := ResolveJWTSecret(context.Background(), repository, "short")
	if err != nil {
		t.Fatalf("ResolveJWTSecret() error = %v", err)
	}
	if secret != "stored-secret" {
		t.Fatalf("secret = %q, want the stored value", secret)
	}
	if len(repository.saved) != 0 {
		t.Fatalf("saved secrets = %v, want none", repository.saved)
	}
}

func TestResolveJWTSecretSeedsDatabaseFromEnvironment(t *testing.T) {
	t.Parallel()

	repository := &fakeJWTSecretRepository{}

	secret, err := ResolveJWTSecret(context.Background(), repository, "  "+testEnvJWTSecret+"  ")
	if err != nil {
		t.Fatalf("ResolveJWTSecret() error = %v", err)
	}
	if secret != testEnvJWTSecret {
		t.Fatalf("secret = %q, want the trimmed environment seed", secret)
	}
	if len(repository.saved) != 1 || repository.saved[0] != testEnvJWTSecret {
		t.Fatalf("saved secrets = %v, want the trimmed environment seed once", repository.saved)
	}
}

func TestResolveJWTSecretUsesConcurrentlyStoredSecret(t *testing.T) {
	t.Parallel()

	const concurrentSecret = "concurrently-stored-secret"
	repository := &fakeJWTSecretRepository{saveResult: concurrentSecret}

	secret, err := ResolveJWTSecret(context.Background(), repository, testEnvJWTSecret)
	if err != nil {
		t.Fatalf("ResolveJWTSecret() error = %v", err)
	}
	if secret != concurrentSecret {
		t.Fatalf("secret = %q, want the concurrently stored %q", secret, concurrentSecret)
	}
}

func TestResolveJWTSecretGeneratesAndPersistsSecret(t *testing.T) {
	t.Parallel()

	repository := &fakeJWTSecretRepository{}

	secret, err := ResolveJWTSecret(context.Background(), repository, "")
	if err != nil {
		t.Fatalf("ResolveJWTSecret() error = %v", err)
	}
	decoded, decodeErr := hex.DecodeString(secret)
	if decodeErr != nil || len(decoded) != jwtSecretEntropyBytes {
		t.Fatalf("generated secret = %q (decoded %d bytes, error %v), want %d hex-encoded bytes", secret, len(decoded), decodeErr, jwtSecretEntropyBytes)
	}
	if len(repository.saved) != 1 || repository.saved[0] != secret {
		t.Fatalf("saved secrets = %v, want the generated secret once", repository.saved)
	}

	again, err := ResolveJWTSecret(context.Background(), repository, "")
	if err != nil {
		t.Fatalf("second ResolveJWTSecret() error = %v", err)
	}
	if again != secret {
		t.Fatalf("second secret = %q, want the persisted %q", again, secret)
	}
}

func TestResolveJWTSecretRejectsShortEnvironmentSeed(t *testing.T) {
	t.Parallel()

	repository := &fakeJWTSecretRepository{}

	_, err := ResolveJWTSecret(context.Background(), repository, strings.Repeat("a", MinJWTSecretLength-1))
	if !errors.Is(err, ErrJWTSecretTooShort) {
		t.Fatalf("ResolveJWTSecret() error = %v, want ErrJWTSecretTooShort", err)
	}
	if len(repository.saved) != 0 {
		t.Fatalf("saved secrets = %v, want none", repository.saved)
	}
}

func TestResolveJWTSecretRejectsBlankStoredSecret(t *testing.T) {
	t.Parallel()

	repository := &fakeJWTSecretRepository{secret: "   ", found: true}

	_, err := ResolveJWTSecret(context.Background(), repository, testEnvJWTSecret)
	if !errors.Is(err, ErrJWTSecretInvalid) {
		t.Fatalf("ResolveJWTSecret() error = %v, want ErrJWTSecretInvalid", err)
	}
}

func TestResolveJWTSecretPropagatesLoadError(t *testing.T) {
	t.Parallel()

	loadErr := errors.New("load failed")
	repository := &fakeJWTSecretRepository{getErr: loadErr}

	_, err := ResolveJWTSecret(context.Background(), repository, testEnvJWTSecret)
	if !errors.Is(err, loadErr) {
		t.Fatalf("ResolveJWTSecret() error = %v, want the load error", err)
	}
}

func TestResolveJWTSecretPropagatesSaveError(t *testing.T) {
	t.Parallel()

	saveErr := errors.New("save failed")
	repository := &fakeJWTSecretRepository{saveErr: saveErr}

	_, err := ResolveJWTSecret(context.Background(), repository, testEnvJWTSecret)
	if !errors.Is(err, saveErr) {
		t.Fatalf("ResolveJWTSecret() error = %v, want the save error", err)
	}
}
