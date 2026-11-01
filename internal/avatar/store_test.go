package avatar

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// pngSignature is the prefix http.DetectContentType keys on to report image/png.
var pngSignature = []byte("\x89PNG\r\n\x1a\n")

func TestSaveRejectsEmptyFile(t *testing.T) {
	t.Parallel()

	if _, err := New(t.TempDir()).Save(nil); !errors.Is(err, ErrEmpty) {
		t.Fatalf("Save(nil) error = %v, want ErrEmpty", err)
	}
}

func TestSaveRejectsUnsupportedImageType(t *testing.T) {
	t.Parallel()

	_, err := New(t.TempDir()).Save([]byte("not an image"))
	if !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("Save(text) error = %v, want ErrUnsupportedType", err)
	}
}

func TestSaveRejectsFileOverTheSizeLimit(t *testing.T) {
	t.Parallel()

	oversized := make([]byte, MaxFileSizeBytes+1)
	copy(oversized, pngSignature)

	if _, err := New(t.TempDir()).Save(oversized); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Save(oversized) error = %v, want ErrTooLarge", err)
	}
}

func TestSaveWritesDetectedImageType(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	store := New(dataDir)

	name, err := store.Save(append(pngSignature, make([]byte, 32)...))
	if err != nil {
		t.Fatalf("Save(png) error = %v", err)
	}
	if !strings.HasSuffix(name, ".png") {
		t.Fatalf("Save(png) name = %q, want a .png suffix", name)
	}

	stored, err := os.Stat(filepath.Join(dataDir, "avatars", name))
	if err != nil {
		t.Fatalf("stat stored avatar: %v", err)
	}
	// Windows reports the ACL-derived 0666 for any file os.WriteFile creates, so the
	// mode only carries the intended 0600 on unix-like systems.
	if runtime.GOOS != "windows" {
		if permission := stored.Mode().Perm(); permission != filePermissions {
			t.Fatalf("stored avatar permission = %o, want %o", permission, filePermissions)
		}
	}
}

// TestPathRejectsNamesOutsideTheDirectory guards the rule that a stored value or a
// URL segment can never address anything but a file directly inside the avatar dir.
func TestPathRejectsNamesOutsideTheDirectory(t *testing.T) {
	t.Parallel()

	store := New(t.TempDir())

	for _, name := range []string{"", "   ", "..", "../escape.png", "nested/avatar.png", `nested\avatar.png`} {
		if _, err := store.Path(name); err == nil {
			t.Fatalf("Path(%q) error = nil, want a rejection", name)
		}
	}
}

func TestRemoveIsIdempotent(t *testing.T) {
	t.Parallel()

	store := New(t.TempDir())
	name, err := store.Save(append(pngSignature, make([]byte, 32)...))
	if err != nil {
		t.Fatalf("Save(png) error = %v", err)
	}

	if err := store.Remove(name); err != nil {
		t.Fatalf("Remove(%q) error = %v", name, err)
	}
	if err := store.Remove(name); err != nil {
		t.Fatalf("second Remove(%q) error = %v, want nil for a missing file", name, err)
	}
}
