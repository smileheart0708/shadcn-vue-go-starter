package migrate

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/uptrace/bun"

	"main/internal/database/dialect/sqlite"
)

const initMigration = `CREATE TABLE items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL
);`

const extendMigration = `ALTER TABLE items ADD COLUMN note TEXT;`

func TestRunAppliesPendingMigrationsOnce(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	ctx := context.Background()
	if err := Run(ctx, db, testMigrations(initMigration, extendMigration)); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	if err := Run(ctx, db, testMigrations(initMigration, extendMigration)); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	if count := countLedgerRows(t, db); count != 2 {
		t.Fatalf("applied migration count = %d, want 2", count)
	}
}

func TestRunRejectsModifiedAppliedMigration(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	ctx := context.Background()
	if err := Run(ctx, db, testMigrations(initMigration)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	err := Run(ctx, db, testMigrations(initMigration+"\n-- edited after release\n"))
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Run() error = %v, want checksum mismatch", err)
	}
}

func TestRunRejectsMigrationMissingFromTheSet(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	ctx := context.Background()
	if err := Run(ctx, db, testMigrations(initMigration, extendMigration)); err != nil {
		t.Fatalf("setup Run() error = %v", err)
	}

	err := Run(ctx, db, fstest.MapFS{
		"0002_generated.sql": &fstest.MapFile{Data: []byte(extendMigration)},
	})
	if err == nil || !strings.Contains(err.Error(), "missing from the migration set") {
		t.Fatalf("Run() error = %v, want the orphaned migration error", err)
	}
}

func TestRunRollsBackFailedMigration(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	ctx := context.Background()
	if err := Run(ctx, db, testMigrations(initMigration)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if err := Run(ctx, db, testMigrations(initMigration, "CREATE TABLE broken (;")); err == nil {
		t.Fatal("Run() error = nil, want a syntax error")
	}
	if count := countLedgerRows(t, db); count != 1 {
		t.Fatalf("applied migration count = %d, want 1 after the failed migration rolled back", count)
	}
}

func TestRunRejectsDuplicateVersions(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	migrations := fstest.MapFS{
		"0001_first.sql":  &fstest.MapFile{Data: []byte(initMigration)},
		"0001_second.sql": &fstest.MapFile{Data: []byte(extendMigration)},
	}
	err := Run(context.Background(), db, migrations)
	if err == nil || !strings.Contains(err.Error(), "duplicate migration version") {
		t.Fatalf("Run() error = %v, want duplicate migration version", err)
	}
}

func TestRunIsIdempotentAcrossConcurrentStartups(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "app.db")
	db1 := openTestDBAt(t, path)
	db2 := openTestDBAt(t, path)
	migrations := testMigrations(initMigration, extendMigration)

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, db := range []*bun.DB{db1, db2} {
		wg.Go(func() {
			<-start
			errs <- Run(context.Background(), db, migrations)
		})
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Run() error = %v", err)
		}
	}
	if count := countLedgerRows(t, db1); count != 2 {
		t.Fatalf("applied migration count = %d, want 2", count)
	}
}

func testMigrations(statements ...string) fstest.MapFS {
	migrations := fstest.MapFS{}
	for index, statement := range statements {
		name := fmt.Sprintf("%04d_generated.sql", index+1)
		migrations[name] = &fstest.MapFile{Data: []byte(statement)}
	}
	return migrations
}

func openTestDB(t *testing.T) *bun.DB {
	t.Helper()
	return openTestDBAt(t, filepath.Join(t.TempDir(), "app.db"))
}

func openTestDBAt(t *testing.T, path string) *bun.DB {
	t.Helper()

	driver := sqlite.New()
	sqldb, err := driver.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	db := bun.NewDB(sqldb, driver.BunDialect())
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close database: %v", closeErr)
		}
	})
	return db
}

func countLedgerRows(t *testing.T, db *bun.DB) int {
	t.Helper()

	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("count applied migrations: %v", err)
	}
	return count
}
