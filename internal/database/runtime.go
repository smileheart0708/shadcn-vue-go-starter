// Package database is the composition root of persistence: it selects a dialect,
// opens it, applies that dialect's migrations and exposes the store the domain
// repositories are built from.
package database

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/uptrace/bun"
	"main/internal/database/dialect"
	"main/internal/database/dialect/sqlite"
	"main/internal/database/migrate"
	"main/internal/database/store"
)

// DriverSQLite is the only dialect compiled into this binary. Adding another one
// means adding its package and a case in selectDriver.
const DriverSQLite = sqlite.Name

type Config struct {
	Driver string
	DSN    string
}

// Runtime is the open database handle. It deliberately exposes only Store: the
// domain services are assembled in main.go from that one value, so this package
// never has to import a domain and adding a domain never edits this file.
type Runtime struct {
	driver string
	db     *bun.DB
	Store  *store.Store
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

	return &Runtime{
		driver: driver.Name(),
		db:     bunDB,
		Store:  store.New(bunDB, driver),
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
