package httpapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The placeholder check sits next to the error-code contract test because both gates
// read the same locale files and `go test ./...` already runs them together.
// Invariants that need nothing from the Go side (key order, `@:` link targets) live in
// web/src/locales/locales.spec.ts instead of being duplicated here.

// placeholderPattern matches the named-interpolation syntax vue-i18n understands.
var placeholderPattern = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)

// TestLocalePlaceholdersMatch guards the one guaranteed gap: vue-i18n silently
// renders a missing named parameter as an empty string (no console warning), so a
// translation that renames `{count}` to `{total}` produces broken copy in exactly
// one language instead of an error. Comparing the declared parameter sets catches
// that at test time. Every other pack is compared against the first one in filename
// order, so no language is special-cased and a single pack (or removing one) needs no
// edit here; placeholder sets must be equal across packs, so the baseline choice does
// not matter.
func TestLocalePlaceholdersMatch(t *testing.T) {
	t.Parallel()

	filenames := localeFiles(t)
	referenceFile := filenames[0]
	referenceMessages := readLocaleMessages(t, referenceFile)

	for _, filename := range filenames[1:] {
		t.Run(filename, func(t *testing.T) {
			t.Parallel()

			for key, message := range readLocaleMessages(t, filename) {
				referenceMessage, ok := referenceMessages[key]
				if !ok {
					// A missing key is already reported by `no-missing-keys-in-other-locales`
					// and by `vue-tsc`; this test only owns the placeholders of shared keys.
					continue
				}

				referencePlaceholders := declaredPlaceholders(referenceMessage)
				placeholders := declaredPlaceholders(message)
				if !slices.Equal(placeholders, referencePlaceholders) {
					t.Errorf(
						"%s placeholders differ: %s%s %s%s",
						key,
						referenceFile,
						formatPlaceholders(referencePlaceholders),
						filename,
						formatPlaceholders(placeholders),
					)
				}
			}
		})
	}
}

// localeFiles lists every message pack in the frontend locale directory. Both gates in
// this package derive from it, so adding or removing a language never requires editing a
// test.
func localeFiles(t *testing.T) []string {
	t.Helper()

	localeDir := filepath.Join("..", "..", "web", "src", "locales")
	entries, err := os.ReadDir(localeDir)
	if err != nil {
		t.Fatalf("read frontend locale dir %s: %v", localeDir, err)
	}

	filenames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		filenames = append(filenames, entry.Name())
	}

	if len(filenames) == 0 {
		t.Fatalf("no locale packs found in %s", localeDir)
	}

	slices.Sort(filenames)

	return filenames
}

// declaredPlaceholders returns the sorted set of parameter names a message
// requires. Repeating a parameter is legal in a translation, so the result is
// deduplicated before comparison.
func declaredPlaceholders(message string) []string {
	matches := placeholderPattern.FindAllStringSubmatch(message, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, match[1])
	}

	slices.Sort(names)

	return slices.Compact(names)
}

func formatPlaceholders(names []string) string {
	if len(names) == 0 {
		return "[]"
	}

	return "[" + strings.Join(names, ",") + "]"
}

// readLocaleMessages flattens a locale file into dotted key to message pairs.
func readLocaleMessages(t *testing.T, filename string) map[string]string {
	t.Helper()

	path := filepath.Join("..", "..", "web", "src", "locales", filename)
	raw, err := os.ReadFile(path) //nolint:gosec // 只读 localeFiles 扫出的文件名拼出的路径
	if err != nil {
		t.Fatalf("read frontend locale file %s: %v", path, err)
	}

	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("decode frontend locale file %s: %v", path, err)
	}

	messages := make(map[string]string)
	flattenLocaleMessages(root, "", messages)
	if len(messages) == 0 {
		t.Fatalf("frontend locale file %s contains no messages", path)
	}

	return messages
}

func flattenLocaleMessages(node map[string]any, prefix string, out map[string]string) {
	for key, value := range node {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		switch typed := value.(type) {
		case string:
			out[path] = typed
		case map[string]any:
			flattenLocaleMessages(typed, path, out)
		}
	}
}
