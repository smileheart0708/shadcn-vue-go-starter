// Package sqlite adapts SQLite for the persistence layer. It is the only place
// that knows about DSN pragmas, the single-writer connection pool, SQLite error
// codes and SQLite-specific SQL.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/schema"
	dialectpkg "main/internal/database/dialect"
	"main/internal/database/dialect/sqlite/migrations"
	sqlitedriver "modernc.org/sqlite"
)

// Name is both the identifier setup persists and the driver name
// modernc.org/sqlite registers with database/sql.
const Name = "sqlite"

const defaultBusyTimeout = 5 * time.Second

// SQLite extended result codes reported through *sqlite.Error for uniqueness
// violations.
const (
	codeConstraintUnique     = 2067 // SQLITE_CONSTRAINT_UNIQUE
	codeConstraintPrimaryKey = 1555 // SQLITE_CONSTRAINT_PRIMARYKEY
)

// Driver implements dialect.Driver for SQLite.
type Driver struct{}

// New returns the SQLite adapter.
func New() Driver {
	return Driver{}
}

// BunDialect reports the bun dialect that renders SQL for SQLite.
func (Driver) BunDialect() schema.Dialect {
	return sqlitedialect.New()
}

// Migrations reports the embedded SQLite migration set.
func (Driver) Migrations() fs.FS {
	return migrations.FS
}

// Open prepares a *sql.DB for SQLite. The DSN keeps working exactly as before
// (a path or a file: URI); pragmas this application depends on are appended.
func (Driver) Open(ctx context.Context, dsn string) (*sql.DB, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, errors.New("database: SQLite DSN is required")
	}

	if dir := parentDir(dsn); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("database: create SQLite data directory: %w", err)
		}
	}

	// modernc.org/sqlite registers itself with database/sql under Name, which is
	// also what this dialect is selected with. Importing the driver is what
	// performs the registration, and it is the same package whose error type
	// UniqueViolation decodes.
	sqldb, err := sql.Open(Name, withPragmas(dsn))
	if err != nil {
		return nil, fmt.Errorf("database: open SQLite: %w", err)
	}
	// WAL allows one writer at a time, so a single connection removes lock
	// contention inside this process and makes the pragma verification below
	// valid for every later query.
	sqldb.SetMaxOpenConns(1)
	sqldb.SetMaxIdleConns(1)
	sqldb.SetConnMaxLifetime(0)
	sqldb.SetConnMaxIdleTime(0)

	if settingsErr := verifySettings(ctx, sqldb); settingsErr != nil {
		closeOnError(sqldb, "verify SQLite settings")
		return nil, settingsErr
	}
	if pingErr := sqldb.PingContext(ctx); pingErr != nil {
		closeOnError(sqldb, "ping SQLite")
		return nil, fmt.Errorf("database: ping SQLite: %w", pingErr)
	}
	return sqldb, nil
}

// ValidateSetupTarget accepts only an empty database or this application's own
// incomplete schema. It runs before migrations so setup cannot take over another
// application's database or a completed installation.
func (Driver) ValidateSetupTarget(ctx context.Context, sqldb *sql.DB) error {
	return dialectpkg.ValidateSetupTarget(ctx, sqldb, listTables)
}

// InspectExisting reports whether the database already holds this application's
// schema without creating or migrating anything. An empty database is not an
// installation; tables this application does not own are rejected so startup
// cannot adopt a foreign database.
func (Driver) InspectExisting(ctx context.Context, dsn string) (bool, error) {
	sqldb, err := sql.Open(Name, dsn)
	if err != nil {
		return false, fmt.Errorf("database: open SQLite for inspection: %w", err)
	}
	defer func() {
		if closeErr := sqldb.Close(); closeErr != nil {
			slog.ErrorContext(ctx, "failed to close SQLite inspection handle", "error", closeErr)
		}
	}()

	tables, err := listTables(ctx, sqldb)
	if err != nil {
		return false, err
	}
	switch dialectpkg.ClassifySetupSchema(tables) {
	case dialectpkg.SetupSchemaEmpty:
		return false, nil
	case dialectpkg.SetupSchemaApplication, dialectpkg.SetupSchemaLedgerOnly:
		return true, nil
	case dialectpkg.SetupSchemaForeign:
	}
	return false, dialectpkg.ErrExistingUnknown
}

func listTables(ctx context.Context, sqldb *sql.DB) (map[string]struct{}, error) {
	rows, err := sqldb.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, fmt.Errorf("database: inspect SQLite tables: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			slog.ErrorContext(ctx, "failed to close SQLite table rows", "error", closeErr)
		}
	}()

	tables := make(map[string]struct{})
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("database: scan SQLite table name: %w", scanErr)
		}
		tables[name] = struct{}{}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("database: read SQLite tables: %w", rowsErr)
	}
	return tables, nil
}

// UniqueViolation decodes the violated unique constraint modernc.org/sqlite
// reports in its message as a comma separated <table>.<column> list. The driver
// exposes no structured constraint name, so this is the one place allowed to
// read the message text.
func (Driver) UniqueViolation(err error) (string, bool) {
	var sqliteErr *sqlitedriver.Error
	if !errors.As(err, &sqliteErr) {
		return "", false
	}
	switch sqliteErr.Code() {
	case codeConstraintUnique, codeConstraintPrimaryKey:
	default:
		return "", false
	}

	const marker = "UNIQUE constraint failed: "
	_, columns, found := strings.Cut(sqliteErr.Error(), marker)
	if !found {
		return "", true
	}
	if end := strings.Index(columns, "("); end >= 0 {
		columns = columns[:end]
	}
	return strings.TrimSpace(columns), true
}

// CaseInsensitiveEqual returns the SQLite case-insensitive equality predicate.
func (Driver) CaseInsensitiveEqual(column string) string {
	return column + " = ? COLLATE NOCASE"
}

// CaseInsensitiveLike returns the SQLite case-insensitive LIKE predicate.
func (Driver) CaseInsensitiveLike(column string) string {
	return column + " LIKE ? COLLATE NOCASE"
}

func withPragmas(dsn string) string {
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

func parentDir(dsn string) string {
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

func verifySettings(ctx context.Context, sqldb *sql.DB) error {
	conn, err := sqldb.Conn(ctx)
	if err != nil {
		return fmt.Errorf("database: get SQLite connection: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.ErrorContext(ctx, "failed to close SQLite connection", "error", closeErr)
		}
	}()

	var foreignKeys int
	if queryErr := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); queryErr != nil {
		return fmt.Errorf("database: verify SQLite foreign keys: %w", queryErr)
	}
	if foreignKeys != 1 {
		return fmt.Errorf("database: expected SQLite foreign keys to be enabled, got %d", foreignKeys)
	}

	var journalMode string
	if queryErr := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); queryErr != nil {
		return fmt.Errorf("database: verify SQLite journal mode: %w", queryErr)
	}
	if !strings.EqualFold(journalMode, "wal") {
		return fmt.Errorf("database: expected SQLite journal mode wal, got %q", journalMode)
	}

	var busyTimeout int64
	if queryErr := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); queryErr != nil {
		return fmt.Errorf("database: verify SQLite busy timeout: %w", queryErr)
	}
	if busyTimeout < defaultBusyTimeout.Milliseconds() {
		return fmt.Errorf("database: expected SQLite busy timeout >= %dms, got %dms", defaultBusyTimeout.Milliseconds(), busyTimeout)
	}
	return nil
}

func closeOnError(sqldb *sql.DB, operation string) {
	if closeErr := sqldb.Close(); closeErr != nil {
		slog.Error("failed to close SQLite database after error", "operation", operation, "error", closeErr)
	}
}
