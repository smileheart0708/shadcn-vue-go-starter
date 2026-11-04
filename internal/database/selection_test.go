package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectionStoreRoundTrip(t *testing.T) {
	t.Parallel()

	store := NewSelectionStore(t.TempDir())
	want := Selection{
		Driver: DriverPostgreSQL,
		DSN:    "postgres://user:secret@example.test/app?sslmode=require",
	}
	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, ok, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !ok {
		t.Fatal("Load() ok = false, want a saved selection")
	}
	if got != want {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
}

func TestSelectionStoreMissingFileIsUnconfigured(t *testing.T) {
	t.Parallel()

	store := NewSelectionStore(t.TempDir())
	_, ok, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if ok {
		t.Fatal("Load() ok = true, want no selection")
	}
}

func TestSelectionStoreRejectsInvalidContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewSelectionStore(dir)
	if err := os.WriteFile(filepath.Join(dir, selectionFileName), []byte(`{"driver":"postgres","dsn":"postgres://user:supersecret@example.test/app"}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, _, err := store.Load()
	if err == nil {
		t.Fatal("Load() error = nil, want invalid driver error")
	}
	if strings.Contains(err.Error(), "supersecret") {
		t.Fatalf("Load() error leaks the DSN: %v", err)
	}
}
