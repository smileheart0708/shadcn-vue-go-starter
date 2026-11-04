package config

import (
	"os"
	"testing"
)

func TestLoadConfigReadsJWTSecretSeedFromEnvironment(t *testing.T) {
	dataDir := t.TempDir()

	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("JWT_SECRET", "env-secret")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.JWTSecret != "env-secret" {
		t.Fatalf("expected JWT secret seed from environment, got %q", cfg.JWTSecret)
	}
	assertEmptyDir(t, dataDir)
}

func TestLoadConfigLeavesJWTSecretSeedEmptyWhenUnset(t *testing.T) {
	dataDir := t.TempDir()

	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("JWT_SECRET", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.JWTSecret != "" {
		t.Fatalf("expected empty JWT secret seed, got %q", cfg.JWTSecret)
	}
	assertEmptyDir(t, dataDir)
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

func assertEmptyDir(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", dir, err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected %q to stay empty, found %d entries", dir, len(entries))
	}
}
