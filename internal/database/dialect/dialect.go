// Package dialect is the only seam between the persistence layer and a concrete
// database product. Everything that differs between databases lives behind
// Driver, so internal/database/store and the domain packages stay dialect
// neutral.
package dialect

import (
	"context"
	"database/sql"
	"io/fs"

	"github.com/uptrace/bun/schema"
)

// Driver adapts one SQL dialect.
//
// Adding PostgreSQL means implementing this interface in a new package and
// shipping that dialect's migration directory. Nothing in
// internal/database/store, internal/database/migrate or the domain packages
// changes.
type Driver interface {
	// Name is the identifier used in configuration, logs and errors.
	Name() string

	// Open turns a DSN into a ready connection pool. Implementations validate
	// the DSN, create missing parent directories, apply pool settings, verify
	// dialect settings and ping the database.
	Open(ctx context.Context, dsn string) (*sql.DB, error)

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
