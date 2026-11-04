package database

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"main/internal/database/dialect"
)

func TestDetectDefaultSQLiteMissingFileIsUnconfigured(t *testing.T) {
	t.Parallel()

	_, ok, err := DetectDefaultSQLite(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("DetectDefaultSQLite() error = %v", err)
	}
	if ok {
		t.Fatal("DetectDefaultSQLite() ok = true, want no installation")
	}
}

func TestDetectDefaultSQLiteEmptyFileIsUnconfigured(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	selection := DefaultSQLiteSelection(dataDir)
	if err := os.WriteFile(selection.DSN, nil, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, ok, err := DetectDefaultSQLite(context.Background(), dataDir)
	if err != nil {
		t.Fatalf("DetectDefaultSQLite() error = %v", err)
	}
	if ok {
		t.Fatal("DetectDefaultSQLite() ok = true, want an empty file to stay unconfigured")
	}
}

func TestDetectDefaultSQLiteAdoptsApplicationDatabase(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	runtime, err := OpenForSetup(context.Background(), Config{
		Driver: DriverSQLite,
		DSN:    DefaultSQLiteSelection(dataDir).DSN,
	})
	if err != nil {
		t.Fatalf("OpenForSetup() error = %v", err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	selection, ok, err := DetectDefaultSQLite(context.Background(), dataDir)
	if err != nil {
		t.Fatalf("DetectDefaultSQLite() error = %v", err)
	}
	if !ok {
		t.Fatal("DetectDefaultSQLite() ok = false, want the default SQLite installation")
	}
	if selection.Driver != DriverSQLite || selection.DSN != DefaultSQLiteSelection(dataDir).DSN {
		t.Fatalf("DetectDefaultSQLite() = %+v, want the derived SQLite selection", selection)
	}
}

func TestDetectDefaultSQLiteRejectsForeignDatabase(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	selection := DefaultSQLiteSelection(dataDir)
	sqldb, err := sql.Open("sqlite", selection.DSN)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if _, err := sqldb.Exec("CREATE TABLE unrelated (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create unrelated table: %v", err)
	}
	if err := sqldb.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	_, ok, err := DetectDefaultSQLite(context.Background(), dataDir)
	if !errors.Is(err, dialect.ErrExistingUnknown) {
		t.Fatalf("DetectDefaultSQLite() error = %v, want ErrExistingUnknown", err)
	}
	if ok {
		t.Fatal("DetectDefaultSQLite() ok = true, want a foreign database rejected")
	}
}
