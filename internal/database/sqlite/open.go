package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const defaultBusyTimeout = 5 * time.Second

// Store is the SQLite adapter for all persistence ports used by this binary.
type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	ctx = normalizeContext(ctx)
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, errors.New("database: SQLite DSN is required")
	}

	if dir := sqliteParentDir(dsn); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("database: create SQLite data directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", sqliteDSN(dsn))
	if err != nil {
		return nil, fmt.Errorf("database: open SQLite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if pragmaErr := verifyPragmas(ctx, db); pragmaErr != nil {
		closeDBOnError(db, "verify SQLite pragmas")
		return nil, pragmaErr
	}
	if pingErr := db.PingContext(ctx); pingErr != nil {
		closeDBOnError(db, "ping SQLite")
		return nil, fmt.Errorf("database: ping SQLite: %w", pingErr)
	}
	if migrateErr := migrate(ctx, db); migrateErr != nil {
		closeDBOnError(db, "run SQLite migrations")
		return nil, migrateErr
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("database: close SQLite: %w", err)
	}
	return nil
}

func sqliteDSN(dsn string) string {
	params := url.Values{}
	params.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", defaultBusyTimeout.Milliseconds()))
	params.Add("_pragma", "foreign_keys(1)")
	params.Add("_pragma", "journal_mode(WAL)")

	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator + params.Encode()
}

func sqliteParentDir(dsn string) string {
	path := dsn
	if before, _, found := strings.Cut(path, "?"); found {
		path = before
	}
	if path == ":memory:" || path == "file::memory:" {
		return ""
	}
	path = strings.TrimPrefix(path, "file:")
	if path == "" || path == ":memory:" {
		return ""
	}
	return filepath.Dir(path)
}

func verifyPragmas(ctx context.Context, db *sql.DB) error {
	conn, connErr := db.Conn(ctx)
	if connErr != nil {
		return fmt.Errorf("database: get SQLite connection: %w", connErr)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.ErrorContext(ctx, "failed to close SQLite connection", "error", closeErr)
		}
	}()

	var foreignKeys int
	if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		return fmt.Errorf("database: verify SQLite foreign keys: %w", err)
	}
	if foreignKeys != 1 {
		return fmt.Errorf("database: expected SQLite foreign keys to be enabled, got %d", foreignKeys)
	}

	var journalMode string
	if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return fmt.Errorf("database: verify SQLite journal mode: %w", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		return fmt.Errorf("database: expected SQLite journal mode wal, got %q", journalMode)
	}

	var busyTimeout int64
	if err := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		return fmt.Errorf("database: verify SQLite busy timeout: %w", err)
	}
	if busyTimeout < defaultBusyTimeout.Milliseconds() {
		return fmt.Errorf("database: expected SQLite busy timeout >= %dms, got %dms", defaultBusyTimeout.Milliseconds(), busyTimeout)
	}

	return nil
}

func closeDBOnError(db *sql.DB, operation string) {
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		slog.Error("failed to close SQLite database after error", "operation", operation, "error", err)
	}
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
