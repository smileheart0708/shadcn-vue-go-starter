package database

import (
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const selectionFileName = "database.json"

// Selection is the durable database choice made during setup. Its DSN may carry
// credentials, so it is written with owner-only permissions and is never part of
// an API response or a log entry.
type Selection struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
}

type SelectionStore struct {
	path string
}

func NewSelectionStore(dataDir string) *SelectionStore {
	return &SelectionStore{path: filepath.Join(dataDir, selectionFileName)}
}

func (s *SelectionStore) Load() (Selection, bool, error) {
	if s == nil {
		return Selection{}, false, errors.New("database: nil selection store")
	}
	data, readErr := os.ReadFile(s.path)
	if errors.Is(readErr, fs.ErrNotExist) {
		return Selection{}, false, nil
	}
	if readErr != nil {
		return Selection{}, false, fmt.Errorf("database: read selection: %w", readErr)
	}

	var selection Selection
	if unmarshalErr := jsonv2.Unmarshal(data, &selection); unmarshalErr != nil {
		return Selection{}, false, fmt.Errorf("database: decode selection: %w", unmarshalErr)
	}
	selection.Driver = strings.ToLower(strings.TrimSpace(selection.Driver))
	selection.DSN = strings.TrimSpace(selection.DSN)
	if validationErr := validateSelection(selection); validationErr != nil {
		return Selection{}, false, validationErr
	}
	return selection, true, nil
}

func (s *SelectionStore) Save(selection Selection) error {
	if s == nil {
		return errors.New("database: nil selection store")
	}
	selection.Driver = strings.ToLower(strings.TrimSpace(selection.Driver))
	selection.DSN = strings.TrimSpace(selection.DSN)
	if validationErr := validateSelection(selection); validationErr != nil {
		return validationErr
	}

	dir := filepath.Dir(s.path)
	if mkdirErr := os.MkdirAll(dir, 0o700); mkdirErr != nil {
		return fmt.Errorf("database: create selection directory: %w", mkdirErr)
	}
	data, marshalErr := jsonv2.Marshal(selection)
	if marshalErr != nil {
		return fmt.Errorf("database: encode selection: %w", marshalErr)
	}

	temp, createErr := os.CreateTemp(dir, ".database-*.tmp")
	if createErr != nil {
		return fmt.Errorf("database: create selection temp file: %w", createErr)
	}
	tempPath := temp.Name()
	defer func() {
		if removeErr := os.Remove(tempPath); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			slog.Warn("failed to remove selection temp file", "path", tempPath, "error", removeErr)
		}
	}()

	if chmodErr := temp.Chmod(0o600); chmodErr != nil {
		closeAfterWriteFailure(temp)
		return fmt.Errorf("database: set selection permissions: %w", chmodErr)
	}
	if _, writeErr := temp.Write(data); writeErr != nil {
		closeAfterWriteFailure(temp)
		return fmt.Errorf("database: write selection: %w", writeErr)
	}
	if syncErr := temp.Sync(); syncErr != nil {
		closeAfterWriteFailure(temp)
		return fmt.Errorf("database: sync selection: %w", syncErr)
	}
	if closeErr := temp.Close(); closeErr != nil {
		return fmt.Errorf("database: close selection: %w", closeErr)
	}
	if renameErr := os.Rename(tempPath, s.path); renameErr != nil {
		return fmt.Errorf("database: replace selection: %w", renameErr)
	}
	return nil
}

func closeAfterWriteFailure(temp *os.File) {
	if closeErr := temp.Close(); closeErr != nil {
		slog.Warn("failed to close selection temporary file", "error", closeErr)
	}
}

func validateSelection(selection Selection) error {
	if _, err := selectDriver(selection.Driver); err != nil {
		return err
	}
	if strings.TrimSpace(selection.DSN) == "" {
		return errors.New("database: selection DSN must not be empty")
	}
	return nil
}
