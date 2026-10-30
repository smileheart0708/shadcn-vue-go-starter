package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
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

func TestUniqueViolationIgnoresUnrelatedErrors(t *testing.T) {
	t.Parallel()

	if constraint, ok := New().UniqueViolation(nil); ok {
		t.Fatalf("UniqueViolation(nil) = (%q, true), want (\"\", false)", constraint)
	}
	if constraint, ok := New().UniqueViolation(context.Canceled); ok {
		t.Fatalf("UniqueViolation(context.Canceled) = (%q, true), want (\"\", false)", constraint)
	}
}
