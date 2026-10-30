// Package migrate applies the SQL migration set of one dialect.
//
// The applied ledger lives in schema_migrations: every applied file is recorded
// with its SHA-256, so a published migration is immutable and either editing one
// byte or dropping one file makes startup fail instead of silently diverging
// from deployed databases. Each migration runs in its own transaction, which
// also makes a concurrent startup idempotent.
package migrate

import (
	"cmp"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

const createLedgerSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    checksum TEXT NOT NULL,
    applied_at INTEGER NOT NULL
)`

// migration is one <version>_<name>.sql file together with its checksum.
type migration struct {
	version  int64
	name     string
	sql      string
	checksum string
}

type ledgerRow struct {
	bun.BaseModel `bun:"table:schema_migrations"`

	Version   int64  `bun:"version,pk"`
	Name      string `bun:"name"`
	Checksum  string `bun:"checksum"`
	AppliedAt int64  `bun:"applied_at"`
}

var errNotApplied = errors.New("database: migration not applied")

// Run applies every migration of fsys that is not applied yet. fsys is expected
// to hold <version>_<name>.sql files at its root.
func Run(ctx context.Context, db *bun.DB, fsys fs.FS) error {
	switch {
	case db == nil:
		return errors.New("database: run migrations with a nil database")
	case fsys == nil:
		return errors.New("database: run migrations with a nil filesystem")
	}

	if err := ensureLedger(ctx, db); err != nil {
		return err
	}
	migrations, err := load(fsys)
	if err != nil {
		return err
	}
	if orphanErr := ensureNoOrphanedMigrations(ctx, db, migrations); orphanErr != nil {
		return orphanErr
	}
	for _, m := range migrations {
		applied, lookupErr := lookup(ctx, db, m.version)
		if lookupErr == nil {
			if reconcileErr := reconcile(m, applied); reconcileErr != nil {
				return reconcileErr
			}
			continue
		}
		if !errors.Is(lookupErr, errNotApplied) {
			return lookupErr
		}
		if applyErr := apply(ctx, db, m); applyErr != nil {
			return applyErr
		}
	}
	return nil
}

func ensureLedger(ctx context.Context, db *bun.DB) error {
	if _, err := db.ExecContext(ctx, createLedgerSQL); err != nil {
		return fmt.Errorf("database: create migration ledger: %w", err)
	}
	return nil
}

// ensureNoOrphanedMigrations fails when the ledger records a migration that the
// migration set no longer contains. Without this check, deleting a published
// file would leave the database on the older schema and no later startup would
// notice.
func ensureNoOrphanedMigrations(ctx context.Context, db *bun.DB, migrations []migration) error {
	known := make(map[int64]struct{}, len(migrations))
	for _, m := range migrations {
		known[m.version] = struct{}{}
	}

	// Column replaces bun's default column list instead of extending it, so only
	// version and name are scanned here and checksum/applied_at stay zero. That is
	// intentional: this check needs nothing else from the ledger.
	var applied []ledgerRow
	if err := db.NewSelect().Model(&applied).Column("version", "name").Scan(ctx); err != nil {
		return fmt.Errorf("database: read applied migrations: %w", err)
	}
	for _, row := range applied {
		if _, ok := known[row.Version]; !ok {
			return fmt.Errorf("database: migration %d (%s) is recorded as applied but missing from the migration set; published migrations are immutable", row.Version, row.Name)
		}
	}
	return nil
}

func load(fsys fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("database: read migrations: %w", err)
	}

	migrations := make([]migration, 0, len(entries))
	seenVersions := make(map[int64]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		version, versionErr := parseVersion(name)
		if versionErr != nil {
			return nil, versionErr
		}
		if existing, exists := seenVersions[version]; exists {
			return nil, fmt.Errorf("database: duplicate migration version %d in %s and %s", version, existing, name)
		}
		content, readErr := fs.ReadFile(fsys, name)
		if readErr != nil {
			return nil, fmt.Errorf("database: read migration %s: %w", name, readErr)
		}
		if strings.TrimSpace(string(content)) == "" {
			return nil, fmt.Errorf("database: migration %s is empty", name)
		}
		checksum := sha256.Sum256(content)
		migrations = append(migrations, migration{
			version:  version,
			name:     name,
			sql:      string(content),
			checksum: hex.EncodeToString(checksum[:]),
		})
		seenVersions[version] = name
	}
	if len(migrations) == 0 {
		return nil, errors.New("database: migration set is empty")
	}

	slices.SortFunc(migrations, func(a, b migration) int {
		return cmp.Compare(a.version, b.version)
	})
	return migrations, nil
}

func parseVersion(name string) (int64, error) {
	prefix, _, found := strings.Cut(name, "_")
	if !found || !strings.HasSuffix(name, ".sql") {
		return 0, fmt.Errorf("database: invalid migration filename %q: expected <version>_<name>.sql", name)
	}
	version, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("database: invalid migration filename %q: version must be a positive integer", name)
	}
	return version, nil
}

func lookup(ctx context.Context, db *bun.DB, version int64) (ledgerRow, error) {
	var applied ledgerRow
	err := db.NewSelect().
		Model(&applied).
		Column("name", "checksum").
		Where("version = ?", version).
		Scan(ctx)
	if err == nil {
		return applied, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ledgerRow{}, errNotApplied
	}
	return ledgerRow{}, fmt.Errorf("database: query migration %d: %w", version, err)
}

func reconcile(m migration, applied ledgerRow) error {
	if applied.Name != m.name {
		return fmt.Errorf("database: migration %d was recorded as %q but is now %q; published migrations are immutable", m.version, applied.Name, m.name)
	}
	if applied.Checksum != m.checksum {
		return fmt.Errorf("database: migration %d (%s) checksum mismatch; published migrations are immutable", m.version, m.name)
	}
	return nil
}

func apply(ctx context.Context, db *bun.DB, m migration) error {
	applyErr := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			return fmt.Errorf("database: apply migration %d (%s): %w", m.version, m.name, err)
		}
		_, err := tx.NewInsert().Model(&ledgerRow{
			Version:   m.version,
			Name:      m.name,
			Checksum:  m.checksum,
			AppliedAt: time.Now().Unix(),
		}).Exec(ctx)
		if err != nil {
			return fmt.Errorf("database: record migration %d (%s): %w", m.version, m.name, err)
		}
		return nil
	})
	if applyErr == nil {
		return nil
	}

	// Another process may have applied and recorded the same migration while
	// this one was running. The ledger decides that, not the driver error text,
	// so the same logic works for every dialect.
	applied, lookupErr := lookup(ctx, db, m.version)
	if lookupErr != nil {
		if errors.Is(lookupErr, errNotApplied) {
			return applyErr
		}
		return errors.Join(applyErr, lookupErr)
	}
	if reconcileErr := reconcile(m, applied); reconcileErr != nil {
		return errors.Join(applyErr, reconcileErr)
	}
	return nil
}
