// Package postgresql adapts PostgreSQL for the persistence layer. It is the only
// place that knows about pgx, PostgreSQL error codes and PostgreSQL-specific SQL.
package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/schema"
	dialectpkg "main/internal/database/dialect"
	"main/internal/database/dialect/postgresql/migrations"
)

const (
	Name = "postgresql"

	maxOpenConns = 10
	maxIdleConns = 5
	connLifetime = 30 * time.Minute
	connIdleTime = 5 * time.Minute

	codeUniqueViolation = "23505"
)

type Driver struct{}

func New() Driver {
	return Driver{}
}

func (Driver) BunDialect() schema.Dialect {
	return pgdialect.New()
}

func (Driver) Migrations() fs.FS {
	return migrations.FS
}

func (Driver) Open(ctx context.Context, dsn string) (*sql.DB, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, errors.New("database: PostgreSQL DSN is required")
	}
	if _, err := pgx.ParseConfig(dsn); err != nil {
		return nil, fmt.Errorf("database: parse PostgreSQL DSN: %w", err)
	}

	sqldb, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("database: open PostgreSQL: %w", err)
	}
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxIdleConns)
	sqldb.SetConnMaxLifetime(connLifetime)
	sqldb.SetConnMaxIdleTime(connIdleTime)

	if pingErr := sqldb.PingContext(ctx); pingErr != nil {
		closeOnError(sqldb, "ping PostgreSQL")
		return nil, fmt.Errorf("database: ping PostgreSQL: %w", pingErr)
	}
	return sqldb, nil
}

func (Driver) ValidateSetupTarget(ctx context.Context, sqldb *sql.DB) error {
	return dialectpkg.ValidateSetupTarget(ctx, sqldb, listTables)
}

func listTables(ctx context.Context, sqldb *sql.DB) (map[string]struct{}, error) {
	rows, err := sqldb.QueryContext(ctx, `
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = current_schema()
	`)
	if err != nil {
		return nil, fmt.Errorf("database: inspect PostgreSQL setup target: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			slog.ErrorContext(ctx, "failed to close PostgreSQL setup target rows", "error", closeErr)
		}
	}()

	tables := make(map[string]struct{})
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("database: scan PostgreSQL setup target: %w", scanErr)
		}
		tables[name] = struct{}{}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("database: read PostgreSQL setup target: %w", rowsErr)
	}
	return tables, nil
}

func (Driver) UniqueViolation(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != codeUniqueViolation {
		return "", false
	}

	switch pgErr.ConstraintName {
	case "users_username_unique_idx":
		return "users.username", true
	case "users_email_unique_idx":
		return "users.email", true
	case "users_owner_unique_idx":
		return "users.role", true
	default:
		return "", true
	}
}

func (Driver) CaseInsensitiveEqual(column string) string {
	return "LOWER(" + column + ") = LOWER(?)"
}

func (Driver) CaseInsensitiveLike(column string) string {
	return "LOWER(" + column + ") LIKE LOWER(?)"
}

func closeOnError(sqldb *sql.DB, operation string) {
	if closeErr := sqldb.Close(); closeErr != nil {
		slog.Error("failed to close PostgreSQL database after error", "operation", operation, "error", closeErr)
	}
}
