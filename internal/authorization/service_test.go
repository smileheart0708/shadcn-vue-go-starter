package authorization

import (
	"testing"

	"main/internal/identity"
)

func TestViewerAuthorizationAppliesDynamicCapabilities(t *testing.T) {
	t.Parallel()

	service := NewService()

	viewer := service.ViewerAuthorization(identity.RoleUser, ViewerOptions{
		AllowSelfDelete: true,
	})

	if service.HasCapability(viewer.Capabilities, CapabilityManagementUsersCreate) {
		t.Fatal("expected regular users to never receive management.users.create")
	}
	if !service.HasCapability(viewer.Capabilities, CapabilityAccountDeleteSelf) {
		t.Fatal("expected account.delete_self capability to be added when allowed")
	}
}

func TestViewerAuthorizationDeniesUnknownRole(t *testing.T) {
	t.Parallel()

	service := NewService()

	viewer := service.ViewerAuthorization("auditor", ViewerOptions{})
	if len(viewer.Capabilities) != 0 {
		t.Fatalf("expected unknown role to receive no capabilities, got %v", viewer.Capabilities)
	}
}
