package dialect

import (
	"context"
	"database/sql"
	"fmt"
)

// SetupSchema classifies the table set of a database that setup targets.
type SetupSchema uint8

const (
	// SetupSchemaEmpty is a database without any table: safe to initialize.
	SetupSchemaEmpty SetupSchema = iota
	// SetupSchemaLedgerOnly is the migration ledger left behind by a failed first
	// migration. The ledger is created outside the per-migration transaction, so
	// this state must stay retryable.
	SetupSchemaLedgerOnly
	// SetupSchemaApplication holds only this application's tables: safe to
	// resume setup or adopt an existing installation.
	SetupSchemaApplication
	// SetupSchemaForeign contains tables this application did not create.
	SetupSchemaForeign
)

// setupTables lists every table created by this application's migrations plus the
// migration ledger. Table names are dialect neutral by contract, so every dialect
// shares this one list; a migration that adds a table must add it here.
// TestMigrationTablesMatchSetupSchema enforces that rule for every dialect.
var setupTables = map[string]struct{}{
	"account_policies":  {},
	"auth_secrets":      {},
	"auth_sessions":     {},
	"credentials":       {},
	"install_state":     {},
	"schema_migrations": {},
	"users":             {},
}

// ClassifySetupSchema decides whether setup may initialize or adopt a database
// with the given table set.
func ClassifySetupSchema(tables map[string]struct{}) SetupSchema {
	if len(tables) == 0 {
		return SetupSchemaEmpty
	}
	for name := range tables {
		if _, known := setupTables[name]; !known {
			return SetupSchemaForeign
		}
	}
	if _, ok := tables["schema_migrations"]; !ok {
		return SetupSchemaForeign
	}
	if len(tables) == 1 {
		return SetupSchemaLedgerOnly
	}
	if _, ok := tables["install_state"]; !ok {
		return SetupSchemaForeign
	}
	return SetupSchemaApplication
}

// ValidateSetupTarget applies the shared setup target policy to the tables the
// dialect reports through listTables. A dialect supplies only the table listing
// query; keeping the decision here means a new rule cannot diverge between
// dialects.
func ValidateSetupTarget(ctx context.Context, db *sql.DB, listTables func(context.Context, *sql.DB) (map[string]struct{}, error)) error {
	tables, err := listTables(ctx, db)
	if err != nil {
		return err
	}
	switch ClassifySetupSchema(tables) {
	case SetupSchemaEmpty, SetupSchemaLedgerOnly:
		return nil
	case SetupSchemaApplication:
	case SetupSchemaForeign:
		return ErrSetupTargetNotEmpty
	}

	var completed bool
	if stateErr := db.QueryRowContext(ctx, "SELECT setup_completed FROM install_state WHERE id = 1").Scan(&completed); stateErr != nil {
		return fmt.Errorf("database: read setup state: %w", stateErr)
	}
	if completed {
		return ErrSetupTargetCompleted
	}
	return nil
}
