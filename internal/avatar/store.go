// Package avatar stores user avatar images on the local filesystem.
//
// It owns everything about where avatars live: the directory layout, the accepted
// image types, the size ceiling and the stored file names. The transport layer only
// reads the multipart body and turns the rejections into API error codes.
package avatar

import (
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

// Remove deletes a stored avatar. A missing file is not an error: cleanup runs after
// the database row already changed, so a stale name must not fail the request.
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

func randomName(extension string) (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("avatar: generate file name: %w", err)
	}

	return hex.EncodeToString(buffer) + extension, nil
}
