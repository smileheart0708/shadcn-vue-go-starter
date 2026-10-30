// Package database is the composition root of persistence: it selects a dialect,
// opens it, applies that dialect's migrations and hands the resulting
// repositories to the domain services.
package database

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/uptrace/bun"
	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/database/dialect"
	"main/internal/database/dialect/sqlite"
	"main/internal/database/migrate"
	"main/internal/database/store"
	"main/internal/identity"
	"main/internal/setup"
)

// DriverSQLite is the only dialect compiled into this binary. Adding another one
// means adding its package and a case in selectDriver.
const DriverSQLite = sqlite.Name

type Config struct {
	Driver string
	DSN    string
}

// Runtime is the composition-root handle for a selected database adapter.
// Domain services receive their own repository instead of this aggregate.
type Runtime struct {
	driver          string
	db              *bun.DB
	Identity        identity.Repository
	Sessions        auth.SessionRepository
	Setup           setup.Repository
	AccountPolicies accountpolicies.Repository
}

func Open(ctx context.Context, config Config) (*Runtime, error) {
	driver, err := selectDriver(config.Driver)
	if err != nil {
		return nil, err
	}

	sqldb, err := driver.Open(ctx, config.DSN)
	if err != nil {
		return nil, err
	}

	bunDB := bun.NewDB(sqldb, driver.BunDialect())
	if migrateErr := migrate.Run(ctx, bunDB, driver.Migrations()); migrateErr != nil {
		closeAfterFailedOpen(bunDB)
		return nil, migrateErr
	}

	repository := store.New(bunDB, driver)
	return &Runtime{
		driver:          driver.Name(),
		db:              bunDB,
		Identity:        repository,
		Sessions:        repository,
		Setup:           repository,
		AccountPolicies: repository,
	}, nil
}

func (r *Runtime) Driver() string {
	return r.driver
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
	case "", sqlite.Name:
		return sqlite.New(), nil
	default:
		return nil, fmt.Errorf("database: driver %q is not compiled into this binary", normalized)
	}
}

func closeAfterFailedOpen(db *bun.DB) {
	if err := db.Close(); err != nil {
		slog.Error("failed to close database after failed startup", "error", err)
	}
}
