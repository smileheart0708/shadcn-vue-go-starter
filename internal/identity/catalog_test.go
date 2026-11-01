package identity

import (
	"slices"
	"testing"
)

// TestStatusesMatchTheServiceVocabulary pins the two properties the generated frontend
// enum depends on: every declared status is exactly the value the service accepts, and
// anything undeclared normalizes to "" (the repository's "no filter").
//
// It cannot prove that a newly declared constant was added to statusOrder — Go cannot
// reflect constants — but it fails the moment normalizeStatus grows a second hand-written
// list that disagrees with the catalog, which is the drift this catalog exists to remove.
func TestStatusesMatchTheServiceVocabulary(t *testing.T) {
	t.Parallel()

	statuses := Statuses()
	if len(statuses) == 0 {
		t.Fatal("expected at least one status")
	}

	for _, status := range statuses {
		if normalizeStatus(status) != status {
			t.Fatalf("status %q is declared but the service does not accept it", status)
		}

		// Query parameters arrive untrimmed, so trim tolerance is part of the contract.
		if normalizeStatus("  "+status+" ") != status {
			t.Fatalf("status %q stopped tolerating surrounding whitespace", status)
		}
	}

	if normalizeStatus("locked") != "" {
		t.Fatal("expected an undeclared status to normalize to the empty string")
	}
}

// TestManagedUserActionsStayInsideTheCatalog guards the direction that breaks the client:
// /api/management/users carries `actions` as a generated enum, so an action the rule can
// produce but the catalog does not declare fails zod parsing and takes the whole listing
// down. A missing entry the other way round only hides the action, which is why only this
// direction is asserted.
func TestManagedUserActionsStayInsideTheCatalog(t *testing.T) {
	t.Parallel()

	catalog := AllManagedUserActions()
	owner := ManagedUserTarget{UserID: 1, Role: RoleOwner, Status: StatusActive}

	for _, status := range Statuses() {
		actions := ManagedUserActions(owner, ManagedUserTarget{UserID: 2, Role: RoleUser, Status: status})
		if len(actions) == 0 {
			t.Fatalf("owner has no action left against a %q target", status)
		}

		for _, action := range actions {
			if !slices.Contains(catalog, action) {
				t.Fatalf("action %q is offered for a %q target but missing from managedUserActionOrder", action, status)
			}
		}
	}
}
