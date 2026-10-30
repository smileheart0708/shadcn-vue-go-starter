package httpapi

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"main/internal/authorization"
	"main/internal/identity"
)

// The empty-capability case is the one that regresses silently: `null` is valid
// JSON but breaks the frontend schema, and a role without capabilities is the
// default for every regular user.
func TestViewerAuthorizationResponseEncodesEmptyCapabilitiesAsArray(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(toViewerAuthorizationResponse(authorization.ViewerAuthorization{
		Role:         identity.RoleUser,
		Capabilities: []authorization.Capability{},
	}))
	if err != nil {
		t.Fatalf("failed to marshal viewer authorization: %v", err)
	}

	if !strings.Contains(string(payload), `"capabilities":[]`) {
		t.Fatalf("expected capabilities to encode as JSON array, got %s", string(payload))
	}
}

func TestViewerAuthorizationResponseSerializesCapabilityStrings(t *testing.T) {
	t.Parallel()

	response := toViewerAuthorizationResponse(authorization.ViewerAuthorization{
		Role: identity.RoleOwner,
		Capabilities: []authorization.Capability{
			authorization.CapabilityManagementUsersRead,
			authorization.CapabilitySystemSettingsUpdate,
		},
	})

	want := []string{"management.users.read", "system.settings.update"}
	if !slices.Equal(response.Capabilities, want) {
		t.Fatalf("expected capabilities %v, got %v", want, response.Capabilities)
	}
	if response.Role != identity.RoleOwner {
		t.Fatalf("expected role %q, got %q", identity.RoleOwner, response.Role)
	}
}

func TestManagedUserActionsToStringsEncodesEmptySlice(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(managedUserActionsToStrings(nil))
	if err != nil {
		t.Fatalf("failed to marshal actions: %v", err)
	}
	if string(payload) != "[]" {
		t.Fatalf("expected empty actions to encode as [], got %s", string(payload))
	}
}
