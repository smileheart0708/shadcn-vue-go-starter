package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigUsesJWTSecretEnvironmentVariable(t *testing.T) {
	dataDir := t.TempDir()
	secretPath := filepath.Join(dataDir, jwtSecretFileName)

	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("JWT_SECRET", "env-secret")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.JWTSecret != "env-secret" {
		t.Fatalf("expected JWT secret from environment, got %q", cfg.JWTSecret)
	}
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Fatalf("expected no persisted JWT secret file when env var is set, stat err=%v", err)
	}
}

func TestLoadConfigUsesPersistedJWTSecretFile(t *testing.T) {
	dataDir := t.TempDir()
	secretPath := filepath.Join(dataDir, jwtSecretFileName)

	if err := os.WriteFile(secretPath, []byte("persisted-secret\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("JWT_SECRET", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.JWTSecret != "persisted-secret" {
		t.Fatalf("expected JWT secret from file, got %q", cfg.JWTSecret)
	}
}

func TestLoadConfigGeneratesAndPersistsJWTSecretWhenMissing(t *testing.T) {
	dataDir := t.TempDir()
	secretPath := filepath.Join(dataDir, jwtSecretFileName)

	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("JWT_SECRET", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.JWTSecret) != jwtSecretEntropyBytes*2 {
		t.Fatalf("expected generated JWT secret length %d, got %d", jwtSecretEntropyBytes*2, len(cfg.JWTSecret))
	}

	storedSecret, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.TrimSpace(string(storedSecret)) != cfg.JWTSecret {
		t.Fatalf("expected persisted JWT secret to match generated secret")
	}

	nextCfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("second LoadConfig() error = %v", err)
	}
	if nextCfg.JWTSecret != cfg.JWTSecret {
		t.Fatalf("expected generated JWT secret to persist across loads")
	}
}

func TestReadJWTSecretFileRejectsNonSecretFilename(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), "other_secret")
	if err := os.WriteFile(filename, []byte("secret"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := readJWTSecretFile(filename); err == nil {
		t.Fatal("expected non-JWT secret filename to be rejected")
	}
}

func TestLoadConfigIgnoresLegacyDatabaseEnvironmentVariables(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("JWT_SECRET", "env-secret")
	t.Setenv("DATABASE_DRIVER", "postgres")
	t.Setenv("DATABASE_DSN", "postgres://user:secret@example.test/app")

	if _, err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
}
