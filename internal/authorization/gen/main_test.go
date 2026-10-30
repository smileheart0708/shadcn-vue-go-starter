package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"main/internal/authorization"
)

// TestGeneratedCapabilitiesAreUpToDate is the drift detector. It runs as part of
// `go test ./...`, which is the only continuous check this repository has, so a
// capability added to the Go catalog without regenerating the frontend module
// fails here instead of silently diverging at runtime.
func TestGeneratedCapabilitiesAreUpToDate(t *testing.T) {
	t.Parallel()

	want, err := render(authorization.All())
	if err != nil {
		t.Fatalf("render capabilities: %v", err)
	}

	path, err := outputPath()
	if err != nil {
		t.Fatalf("resolve output path: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("%s is stale; run: go generate ./internal/authorization", path)
	}
}

func TestRenderRejectsEmptyCatalog(t *testing.T) {
	t.Parallel()

	if _, err := render(nil); err == nil {
		t.Fatal("expected an empty catalog to be rejected")
	}
}

func TestRenderRejectsIdentifierCollisions(t *testing.T) {
	t.Parallel()

	_, err := render([]authorization.Capability{"system.settings.read", "system.settings_read"})
	if err == nil {
		t.Fatal("expected colliding capability identifiers to be rejected")
	}
	if !strings.Contains(err.Error(), "systemSettingsRead") {
		t.Fatalf("expected the collision message to name the identifier, got %v", err)
	}
}

func TestTSIdentifier(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		capability string
		want       string
	}{
		{capability: "system.settings.read", want: "systemSettingsRead"},
		{capability: "system.settings.update", want: "systemSettingsUpdate"},
		{capability: "management.users.read", want: "managementUsersRead"},
		{capability: "management.system_logs.read", want: "managementSystemLogsRead"},
		{capability: "account.delete_self", want: "accountDeleteSelf"},
	} {
		got, err := tsIdentifier(tc.capability)
		if err != nil {
			t.Fatalf("tsIdentifier(%q) failed: %v", tc.capability, err)
		}
		if got != tc.want {
			t.Fatalf("tsIdentifier(%q): expected %q, got %q", tc.capability, tc.want, got)
		}
	}
}

func TestTSIdentifierRejectsUnusableKeys(t *testing.T) {
	t.Parallel()

	for _, capability := range []string{"", ".", "2fa.read", "system settings/read", "system.\nread"} {
		if _, err := tsIdentifier(capability); err == nil {
			t.Fatalf("expected %q to be rejected", capability)
		}
	}
}
