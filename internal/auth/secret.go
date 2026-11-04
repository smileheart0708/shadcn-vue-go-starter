package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	// MinJWTSecretLength is the shortest JWT_SECRET seed accepted. The generated
	// secret is always longer; this bound only guards operator-provided values.
	MinJWTSecretLength    = 32
	jwtSecretEntropyBytes = 64
)

var (
	ErrJWTSecretNotFound = errors.New("auth: JWT secret not found")
	ErrJWTSecretTooShort = errors.New("auth: JWT secret must be at least 32 characters")
	ErrJWTSecretInvalid  = errors.New("auth: stored JWT secret is empty")
)

// JWTSecretRepository persists the single signing secret of an installation.
type JWTSecretRepository interface {
	GetJWTSecret(ctx context.Context) (string, error)
	// EnsureJWTSecret stores candidate when no row exists and returns the
	// effective stored value. When a concurrent starter won the insert, it
	// returns that secret instead.
	EnsureJWTSecret(ctx context.Context, candidate string) (string, error)
}

// ResolveJWTSecret returns the effective signing secret. A stored value always
// wins; a missing row is filled from envSecret, or from a generated secret when
// the seed is blank. The chosen value is persisted before it is returned, so
// every later start reads the same secret.
func ResolveJWTSecret(ctx context.Context, repository JWTSecretRepository, envSecret string) (string, error) {
	stored, err := repository.GetJWTSecret(ctx)
	if err == nil {
		if strings.TrimSpace(stored) == "" {
			return "", ErrJWTSecretInvalid
		}
		return stored, nil
	}
	if !errors.Is(err, ErrJWTSecretNotFound) {
		return "", fmt.Errorf("auth: load JWT secret: %w", err)
	}

	secret := strings.TrimSpace(envSecret)
	switch {
	case secret == "":
		secret, err = generateJWTSecret()
		if err != nil {
			return "", err
		}
	case len(secret) < MinJWTSecretLength:
		return "", ErrJWTSecretTooShort
	}

	effective, saveErr := repository.EnsureJWTSecret(ctx, secret)
	if saveErr != nil {
		return "", fmt.Errorf("auth: persist JWT secret: %w", saveErr)
	}
	return effective, nil
}

func generateJWTSecret() (string, error) {
	secret := make([]byte, jwtSecretEntropyBytes)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("auth: generate JWT secret: %w", err)
	}
	return hex.EncodeToString(secret), nil
}
