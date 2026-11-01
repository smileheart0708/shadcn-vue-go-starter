package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"main/internal/authorization"
	"main/internal/identity"
)

// TestGeneratedModulesAreUpToDate is the drift detector. It runs as part of
// `go test ./...`, which is the only continuous check this repository has, so a catalog
// change without regenerating the frontend modules fails here instead of silently
// diverging at runtime.
func TestGeneratedModulesAreUpToDate(t *testing.T) {
	t.Parallel()

	root, err := repositoryRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	files, err := renderAll()
	if err != nil {
		t.Fatalf("render modules: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one generated module")
	}

	for _, file := range files {
		t.Run(file.relativePath, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(root, filepath.FromSlash(file.relativePath))
			committed, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read %s: %v", path, readErr)
			}

			if !bytes.Equal(committed, file.payload) {
				t.Fatalf("%s is stale; run: go generate ./internal/contractgen", path)
			}
		})
	}
}

func TestRenderCapabilitiesRejectsEmptyCatalog(t *testing.T) {
	t.Parallel()

	if _, err := renderCapabilities(nil); err == nil {
		t.Fatal("expected an empty catalog to be rejected")
	}
}

func TestRenderContractRejectsEmptyCatalogs(t *testing.T) {
	t.Parallel()

	roles := []identity.Role{{Key: identity.RoleOwner, Rank: 100}}
	statuses := []string{identity.StatusActive}
	actions := []identity.ManagedUserAction{identity.ManagedUserActionUpdate}

	tests := []struct {
		name     string
		roles    []identity.Role
		statuses []string
		actions  []identity.ManagedUserAction
	}{
		{name: "no roles", roles: nil, statuses: statuses, actions: actions},
		{name: "no statuses", roles: roles, statuses: nil, actions: actions},
		{name: "no actions", roles: roles, statuses: statuses, actions: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, err := renderContract(test.roles, test.statuses, test.actions); err == nil {
				t.Fatal("expected an empty catalog to be rejected")
			}
		})
	}
}

func TestRenderRejectsIdentifierCollisions(t *testing.T) {
	t.Parallel()

	_, err := renderCapabilities([]authorization.Capability{"system.settings.read", "system.settings_read"})
	if err == nil {
		t.Fatal("expected colliding capability identifiers to be rejected")
	}
	if !strings.Contains(err.Error(), "systemSettingsRead") {
		t.Fatalf("expected the collision message to name the identifier, got %v", err)
	}

	_, err = renderContract(
		[]identity.Role{{Key: "system_admin", Rank: 100}, {Key: "system.admin", Rank: 50}},
		[]string{identity.StatusActive},
		[]identity.ManagedUserAction{identity.ManagedUserActionUpdate},
	)
	if err == nil {
		t.Fatal("expected colliding role identifiers to be rejected")
	}
	if !strings.Contains(err.Error(), "systemAdmin") {
		t.Fatalf("expected the collision message to name the identifier, got %v", err)
	}
}

// TestRenderContractKeepsDeclarationOrder guards why the Go catalogs are slices and not
// maps: the generated enum follows declaration order, so the diff of a new role stays
// reviewable.
func TestRenderContractKeepsDeclarationOrder(t *testing.T) {
	t.Parallel()

	payload, err := renderContract(
		[]identity.Role{{Key: "auditor", Rank: 60}, {Key: "support", Rank: 40}},
		[]string{identity.StatusActive, identity.StatusDisabled},
		[]identity.ManagedUserAction{identity.ManagedUserActionUpdate},
	)
	if err != nil {
		t.Fatalf("render contract: %v", err)
	}

	rendered := string(payload)
	auditor := strings.Index(rendered, "'auditor'")
	support := strings.Index(rendered, "'support'")
	if auditor < 0 || support < 0 {
		t.Fatalf("expected both roles in the rendered module, got:\n%s", rendered)
	}
	if auditor > support {
		t.Fatal("expected the rendered enum to follow declaration order")
	}
}

func TestTSIdentifier(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		value string
		want  string
	}{
		{value: "system.settings.read", want: "systemSettingsRead"},
		{value: "management.system_logs.read", want: "managementSystemLogsRead"},
		{value: "account.delete_self", want: "accountDeleteSelf"},
		{value: "owner", want: "owner"},
		{value: "managed-user.action", want: "managedUserAction"},
	} {
		got, err := tsIdentifier(tc.value)
		if err != nil {
			t.Fatalf("tsIdentifier(%q) failed: %v", tc.value, err)
		}
		if got != tc.want {
			t.Fatalf("tsIdentifier(%q): expected %q, got %q", tc.value, tc.want, got)
		}
	}
}

func TestTSIdentifierRejectsUnusableKeys(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", ".", "2fa.read", "system settings/read", "system.\nread", "it's"} {
		if _, err := tsIdentifier(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

// TestTSStringLiteralRejectsUnescapableValues keeps the byte-for-byte comparison honest:
// a value that needs escaping would render a different shape than the review assumes.
func TestTSStringLiteralRejectsUnescapableValues(t *testing.T) {
	t.Parallel()

	if literal, err := tsStringLiteral("active"); err != nil || literal != "'active'" {
		t.Fatalf("expected a plain literal, got %q (err %v)", literal, err)
	}

	for _, value := range []string{"", "it's", `back\slash`, `quo"te`, "line\nbreak"} {
		if _, err := tsStringLiteral(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}
