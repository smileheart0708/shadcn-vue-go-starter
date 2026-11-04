package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	dialectpkg "main/internal/database/dialect"
)

func TestOpenAppliesPragmasAndSingleConnectionPool(t *testing.T) {
	t.Parallel()

	sqldb, err := New().Open(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := sqldb.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	var foreignKeys int
	if queryErr := sqldb.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); queryErr != nil {
		t.Fatalf("query foreign_keys: %v", queryErr)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	var busyTimeout int64
	if queryErr := sqldb.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); queryErr != nil {
		t.Fatalf("query busy_timeout: %v", queryErr)
	}
	if busyTimeout < defaultBusyTimeout.Milliseconds() {
		t.Fatalf("busy_timeout = %dms, want at least %dms", busyTimeout, defaultBusyTimeout.Milliseconds())
	}

	var journalMode string
	if queryErr := sqldb.QueryRow("PRAGMA journal_mode").Scan(&journalMode); queryErr != nil {
		t.Fatalf("query journal_mode: %v", queryErr)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	if stats := sqldb.Stats(); stats.MaxOpenConnections != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", stats.MaxOpenConnections)
	}
}

func TestOpenRejectsEmptyDSN(t *testing.T) {
	t.Parallel()

	if _, err := New().Open(context.Background(), "   "); err == nil {
		t.Fatal("Open() error = nil, want an error for an empty DSN")
	}
}

func TestValidateSetupTargetAcceptsEmptyDatabase(t *testing.T) {
	t.Parallel()

	driver := New()
	sqldb, err := driver.Open(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := sqldb.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	if err := driver.ValidateSetupTarget(context.Background(), sqldb); err != nil {
		t.Fatalf("ValidateSetupTarget() error = %v", err)
	}
}

func TestValidateSetupTargetRejectsForeignTable(t *testing.T) {
	t.Parallel()

	driver := New()
	sqldb, err := driver.Open(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := sqldb.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})
	if _, err := sqldb.Exec("CREATE TABLE unrelated (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create unrelated table: %v", err)
	}

	if err := driver.ValidateSetupTarget(context.Background(), sqldb); !errors.Is(err, dialectpkg.ErrSetupTargetNotEmpty) {
		t.Fatalf("ValidateSetupTarget() error = %v, want ErrSetupTargetNotEmpty", err)
	}
}

func TestUniqueViolationIgnoresUnrelatedErrors(t *testing.T) {
	t.Parallel()

	if constraint, ok := New().UniqueViolation(nil); ok {
		t.Fatalf("UniqueViolation(nil) = (%q, true), want (\"\", false)", constraint)
	}
	if constraint, ok := New().UniqueViolation(context.Canceled); ok {
		t.Fatalf("UniqueViolation(context.Canceled) = (%q, true), want (\"\", false)", constraint)
	}
}
