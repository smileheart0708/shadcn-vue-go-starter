// Package avatar stores user avatar images on the local filesystem.
//
// It owns everything about where avatars live: the directory layout, the accepted
// image types, the size ceiling and the stored file names. The transport layer only
// reads the multipart body and turns the rejections into API error codes.
package avatar

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// MaxFileSizeBytes is the largest avatar the API accepts. It mirrors
// MAX_AVATAR_FILE_SIZE_BYTES in web/src/lib/avatar.ts.
const MaxFileSizeBytes int64 = 5 * 1024 * 1024

const (
	dirPermissions  = 0o750
	filePermissions = 0o600
	randomNameBytes = 16
)

// Rejections a caller must translate into its own response vocabulary. They are
// errors rather than API codes so this package stays transport-agnostic.
var (
	ErrEmpty           = errors.New("avatar: file is empty")
	ErrTooLarge        = errors.New("avatar: file exceeds the size limit")
	ErrUnsupportedType = errors.New("avatar: unsupported image type")
)

// allowedTypes maps the content type http.DetectContentType reports to the
// extension the file is stored with. Keep it in sync with
// SUPPORTED_AVATAR_FILE_TYPES in web/src/lib/avatar.ts.
var allowedTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// Store keeps avatars as flat files in a single directory.
type Store struct {
	dir string
}

func New(dataDir string) *Store {
	return &Store{dir: filepath.Join(dataDir, "avatars")}
}

// Save validates data and writes it under a random name. The returned name is what
// the identity domain stores in the user row and what Path/Remove later accept.
func (s *Store) Save(data []byte) (string, error) {
	if len(data) == 0 {
		return "", ErrEmpty
	}
	if int64(len(data)) > MaxFileSizeBytes {
		return "", ErrTooLarge
	}

	// Sniff from the bytes, never from the client-supplied filename or content type.
	detectedType := http.DetectContentType(data[:min(512, len(data))])
	extension, supported := allowedTypes[detectedType]
	if !supported {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedType, detectedType)
	}

	name, err := randomName(extension)
	if err != nil {
		return "", err
	}

	if mkdirErr := os.MkdirAll(s.dir, dirPermissions); mkdirErr != nil {
		return "", fmt.Errorf("avatar: create directory %q: %w", s.dir, mkdirErr)
	}

	fullPath, err := s.Path(name)
	if err != nil {
		return "", err
	}
	if writeErr := os.WriteFile(fullPath, data, filePermissions); writeErr != nil {
		return "", fmt.Errorf("avatar: write %q: %w", name, writeErr)
	}

	return name, nil
}

// Path resolves a stored name to a full path inside the avatar directory.
//
// This is the only place that validates a caller-supplied name: a stored value or a
// URL segment can never escape the directory because only bare file names pass.
func (s *Store) Path(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || trimmed != filepath.Base(trimmed) || strings.Contains(trimmed, `\`) {
		return "", fmt.Errorf("avatar: invalid file name %q", name)
	}

	cleanedDir := filepath.Clean(s.dir)
	fullPath := filepath.Join(cleanedDir, trimmed)
	relativePath, err := filepath.Rel(cleanedDir, fullPath)
	if err != nil {
		return "", fmt.Errorf("avatar: resolve %q: %w", name, err)
	}
	if relativePath == "." || relativePath == "" || strings.HasPrefix(relativePath, "..") || filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("avatar: file name %q escapes the avatar directory", name)
	}

	return fullPath, nil
}

// Remove deletes a stored avatar. A missing file is not an error: callers run cleanup
// after the database row changed or the upload failed, so a stale name must not fail
// the request.
func (s *Store) Remove(name string) error {
	fullPath, err := s.Path(name)
	if err != nil {
		return err
	}
	if removeErr := os.Remove(fullPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return fmt.Errorf("avatar: remove %q: %w", name, removeErr)
	}

	return nil
}

// ReferenceLister reports the stored avatar names user rows still reference. Paths
// are reduced to their base name, so a legacy value cannot hide a live file.
type ReferenceLister func(ctx context.Context) ([]string, error)

// Reconcile removes stored avatars that listReferences does not report. The database
// is the source of truth and files are derived, so an unreferenced file is left over
// from a failed removal, an interrupted upload, or a database that was reset or
// restored. A lister error aborts the sweep before anything is deleted.
//
// Callers must run it while no upload can be in flight. main.go does so before the
// API with the upload route is published, which also covers setup completion. Calling
// it on a serving path would race Save — the file is written before the row records
// it — and delete a live upload, so gate on file mtime or a lease first if that ever
// changes.
func (s *Store) Reconcile(ctx context.Context, listReferences ReferenceLister) (int, error) {
	paths, err := listReferences(ctx)
	if err != nil {
		return 0, fmt.Errorf("avatar: list references: %w", err)
	}
	referenced := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		referenced[filepath.Base(path)] = struct{}{}
	}

	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("avatar: read directory: %w", err)
	}

	var removed int
	var failures []error
	for _, entry := range entries {
		name := entry.Name()
		// IsRegular also excludes directories and symlinks, so the sweep never
		// deletes anything but the flat files Save writes.
		if !isStoredName(name) || !entry.Type().IsRegular() {
			continue
		}
		if _, inUse := referenced[name]; inUse {
			continue
		}
		if removeErr := s.Remove(name); removeErr != nil {
			failures = append(failures, removeErr)
			continue
		}
		removed++
	}
	return removed, errors.Join(failures...)
}

// isStoredName recognizes only names Save can produce, so a directory sweep never
// touches files this package did not create.
func isStoredName(name string) bool {
	extension := filepath.Ext(name)
	known := false
	for _, allowed := range allowedTypes {
		if allowed == extension {
			known = true
			break
		}
	}
	if !known {
		return false
	}

	stem := strings.TrimSuffix(name, extension)
	if len(stem) != 2*randomNameBytes {
		return false
	}
	for _, char := range stem {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func randomName(extension string) (string, error) {
	buffer := make([]byte, randomNameBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("avatar: generate file name: %w", err)
	}

	return hex.EncodeToString(buffer) + extension, nil
}
