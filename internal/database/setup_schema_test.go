package database

import (
	"io/fs"
	"maps"
	"regexp"
	"testing"

	"main/internal/database/dialect"
	"main/internal/database/dialect/postgresql"
	"main/internal/database/dialect/sqlite"
)

var createTablePattern = regexp.MustCompile(`(?im)^\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:"([^"]+)"|([a-zA-Z_][a-zA-Z0-9_]*))`)

// TestMigrationTablesMatchSetupSchema keeps dialect.setupTables aligned with the
// append-only migrations. Without it, a new table makes ClassifySetupSchema report
// a migrated database as foreign, so every existing SQLite installation fails to
// start.
func TestMigrationTablesMatchSetupSchema(t *testing.T) {
	t.Parallel()

	sqliteTables := migrationTables(t, "sqlite", sqlite.New().Migrations())
	postgresTables := migrationTables(t, "postgresql", postgresql.New().Migrations())
	if !maps.Equal(sqliteTables, postgresTables) {
		t.Fatalf("dialect migration tables differ: sqlite=%v postgresql=%v", sqliteTables, postgresTables)
	}

	sqliteTables["schema_migrations"] = struct{}{}
	if got := dialect.ClassifySetupSchema(sqliteTables); got != dialect.SetupSchemaApplication {
		t.Fatalf("ClassifySetupSchema(migrated tables) = %d, want SetupSchemaApplication; add the new table to dialect.setupTables", got)
	}
}

func migrationTables(t *testing.T, dialectName string, fsys fs.FS) map[string]struct{} {
	t.Helper()

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("read %s migrations: %v", dialectName, err)
	}
	tables := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, readErr := fs.ReadFile(fsys, entry.Name())
		if readErr != nil {
			t.Fatalf("read %s migration %s: %v", dialectName, entry.Name(), readErr)
		}
		for _, match := range createTablePattern.FindAllStringSubmatch(string(content), -1) {
			name := match[1]
			if name == "" {
				name = match[2]
			}
			tables[name] = struct{}{}
		}
	}
	if len(tables) == 0 {
		t.Fatalf("%s migrations: no CREATE TABLE statement matched", dialectName)
	}
	return tables
}
