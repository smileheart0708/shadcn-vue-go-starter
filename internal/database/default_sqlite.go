package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"main/internal/database/dialect"
	"main/internal/database/dialect/sqlite"
)

const defaultSQLiteFileName = "data.db"

// DefaultSQLiteSelection is the implicit SQLite choice derived from the data
// directory. SQLite carries no credentials, so it never needs a selection file;
// only external databases do.
func DefaultSQLiteSelection(dataDir string) Selection {
	return Selection{Driver: DriverSQLite, DSN: filepath.Join(dataDir, defaultSQLiteFileName)}
}

// DetectDefaultSQLite adopts the default SQLite database when it already holds
// this application's schema. A missing or empty file means no installation; a
// file owned by another application is an error so startup fails loudly instead
// of offering setup that could never succeed.
func DetectDefaultSQLite(ctx context.Context, dataDir string) (Selection, bool, error) {
	selection := DefaultSQLiteSelection(dataDir)
	info, err := os.Stat(selection.DSN)
	if errors.Is(err, fs.ErrNotExist) {
		return Selection{}, false, nil
	}
	if err != nil {
		return Selection{}, false, fmt.Errorf("database: inspect default SQLite database: %w", err)
	}
	if info.IsDir() {
		return Selection{}, false, fmt.Errorf("database: default SQLite path %q is a directory", selection.DSN)
	}

	existing, inspectErr := sqlite.New().InspectExisting(ctx, selection.DSN)
	if inspectErr != nil {
		if errors.Is(inspectErr, dialect.ErrExistingUnknown) {
			return Selection{}, false, fmt.Errorf("%w: %s (remove or rename the file to start a new installation)", inspectErr, selection.DSN)
		}
		return Selection{}, false, fmt.Errorf("database: inspect default SQLite database %q: %w", selection.DSN, inspectErr)
	}
	if !existing {
		return Selection{}, false, nil
	}
	return selection, true, nil
}
