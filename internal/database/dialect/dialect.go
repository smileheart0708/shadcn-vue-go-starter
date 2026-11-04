// Package dialect is the only seam between the persistence layer and a concrete
// database product. Everything that differs between databases lives behind
// Driver, so internal/database/store and the domain packages stay dialect
// neutral.
package dialect

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"

	"github.com/uptrace/bun/schema"
)

var (
	ErrSetupTargetNotEmpty  = errors.New("database: setup target is not an empty application database")
	ErrSetupTargetCompleted = errors.New("database: setup target already completed this application")
	ErrExistingUnknown      = errors.New("database: existing database was not created by this application")
)

// Driver adapts one SQL dialect.
//
// Adding another dialect means implementing this interface in a new package,
// shipping that dialect's migration directory and exposing it through setup.
// Nothing in internal/database/store, internal/database/migrate or the domain
// packages changes.
type Driver interface {
	// Open turns a DSN into a ready connection pool. Implementations validate
	// the DSN, create missing parent directories, apply pool settings, verify
	// dialect settings and ping the database.
	Open(ctx context.Context, dsn string) (*sql.DB, error)

	// ValidateSetupTarget checks that a database may be initialized by setup:
	// it is either empty or contains only this application's incomplete schema.
	// Normal startup does not call it; setup does, before applying migrations.
	ValidateSetupTarget(ctx context.Context, db *sql.DB) error

	// Migrations returns the embedded migration set of this dialect, rooted at
	// the directory holding <version>_<name>.sql files.
	Migrations() fs.FS

	// BunDialect returns the bun dialect that renders SQL for this database.
	BunDialect() schema.Dialect

	// UniqueViolation reports which unique constraint err violated as a comma
	// separated <table>.<column> list, for example "users.username". A dialect
	// whose driver only names the index has to resolve it back to those columns.
	// Decoding driver-specific errors is the dialect's job; callers only match
	// the returned column names.
	UniqueViolation(err error) (constraint string, ok bool)

	// CaseInsensitiveEqual returns a SQL predicate comparing column against one
	// placeholder, ignoring case.
	CaseInsensitiveEqual(column string) string

	// CaseInsensitiveLike returns a SQL predicate matching column against one
	// LIKE pattern placeholder, ignoring case.
	CaseInsensitiveLike(column string) string
}
