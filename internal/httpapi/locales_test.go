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

// The frontend locale files are validated from here because the repository has no
// frontend test runner, and `go test ./...` is already the gate that reads them
// (see TestErrorCodesMatchFrontendLocale). Keeping the locale invariants in one
// place beats adding a second toolchain just to compare two JSON files.

// placeholderPattern matches the named-interpolation syntax vue-i18n understands.
var placeholderPattern = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)

// TestLocalePlaceholdersMatch guards the one guaranteed gap: vue-i18n silently
// renders a missing named parameter as an empty string (no console warning), so a
// translation that renames `{count}` to `{total}` produces broken copy in exactly
// one language instead of an error. Comparing the declared parameter sets catches
// that at test time.
func TestLocalePlaceholdersMatch(t *testing.T) {
	t.Parallel()

	zhMessages := readLocaleMessages(t, "zh-CN.json")
	enMessages := readLocaleMessages(t, "en-US.json")

	for key, zhMessage := range zhMessages {
		enMessage, ok := enMessages[key]
		if !ok {
			// A missing key is already reported by `no-missing-keys-in-other-locales`
			// and by `vue-tsc`; this test only owns the placeholders of shared keys.
			continue
		}

		zhPlaceholders := declaredPlaceholders(zhMessage)
		enPlaceholders := declaredPlaceholders(enMessage)
		if !slices.Equal(zhPlaceholders, enPlaceholders) {
			t.Errorf(
				"%s placeholders differ: zh-CN%s en-US%s",
				key,
				formatPlaceholders(zhPlaceholders),
				formatPlaceholders(enPlaceholders),
			)
		}
	}
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
	raw, err := os.ReadFile(path)
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
