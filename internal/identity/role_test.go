package identity

import "testing"

// IsHighestRole is what object-level rules use instead of comparing against RoleOwner.
// Unknown and empty roles are never top-level; the declaration order itself is guarded by
// TestRolesAreOrderedByDescendingRank in management_test.go.
func TestIsHighestRole(t *testing.T) {
	t.Parallel()

	if !IsHighestRole(RoleOwner) {
		t.Fatal("expected owner to hold the highest rank")
	}
	if IsHighestRole(RoleUser) {
		t.Fatal("expected user not to hold the highest rank")
	}

	if IsHighestRole("auditor") {
		t.Fatal("expected unknown role not to hold the highest rank")
	}
	if IsHighestRole("") {
		t.Fatal("expected empty role not to hold the highest rank")
	}
}
