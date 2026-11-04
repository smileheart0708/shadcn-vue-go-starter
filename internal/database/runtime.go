// Package database is the composition root of persistence: it selects a dialect,
// opens it, applies that dialect's migrations and exposes the store the domain
// repositories are built from.
package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/uptrace/bun"
	"main/internal/database/dialect"
	"main/internal/database/dialect/postgresql"
	"main/internal/database/dialect/sqlite"
	"main/internal/database/migrate"
	"main/internal/database/store"
)

const (
	DriverSQLite     = sqlite.Name
	DriverPostgreSQL = postgresql.Name
)

var (
	ErrConnect           = errors.New("database: connect failed")
	ErrMigrate           = errors.New("database: migrate failed")
	ErrUnsupportedDriver = errors.New("database: unsupported driver")
)

type Config struct {
	Driver string
	DSN    string
}

// Runtime is the open database handle. It deliberately exposes only Store: the
// domain services are assembled in main.go from that one value, so this package
// never has to import a domain and adding a domain never edits this file.
type Runtime struct {
	db    *bun.DB
	Store *store.Store
}

func Open(ctx context.Context, config Config) (*Runtime, error) {
	return open(ctx, config, false)
}

// OpenForSetup validates that the target may be initialized before applying
// migrations. Normal startup uses Open because an existing application database
// is expected there.
func OpenForSetup(ctx context.Context, config Config) (*Runtime, error) {
	return open(ctx, config, true)
}

func open(ctx context.Context, config Config, validateSetupTarget bool) (*Runtime, error) {
	driver, err := selectDriver(config.Driver)
	if err != nil {
		return nil, err
	}

	sqldb, err := driver.Open(ctx, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConnect, err)
	}

	bunDB := bun.NewDB(sqldb, driver.BunDialect())
	if validateSetupTarget {
		if validateErr := driver.ValidateSetupTarget(ctx, sqldb); validateErr != nil {
			closeAfterFailedOpen(bunDB)
			return nil, validateErr
		}
	}
	if migrateErr := migrate.Run(ctx, bunDB, driver.Migrations()); migrateErr != nil {
		closeAfterFailedOpen(bunDB)
		return nil, fmt.Errorf("%w: %w", ErrMigrate, migrateErr)
	}

	return &Runtime{
		db:    bunDB,
		Store: store.New(bunDB, driver),
	}, nil
}

func (r *Runtime) Close() error {
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("database: close runtime: %w", err)
	}
	return nil
}

func selectDriver(name string) (dialect.Driver, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch normalized {
	case sqlite.Name:
		return sqlite.New(), nil
	case postgresql.Name:
		return postgresql.New(), nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedDriver, normalized)
	}
}

func closeAfterFailedOpen(db *bun.DB) {
	if err := db.Close(); err != nil {
		slog.Error("failed to close database after failed startup", "error", err)
	}
}
