package authorization

import (
	"slices"
	"testing"

	"main/internal/identity"
)

// expectedPolicyDerived is the set of capabilities that are granted from
// account_policies at request time instead of the role matrix. It is spelled out
// here rather than in production code so the test fails when a capability is
// declared but simply forgotten by both layers.
var expectedPolicyDerived = []Capability{
	CapabilityAccountDeleteSelf,
}

func TestCapabilityCatalogHasNoDuplicates(t *testing.T) {
	t.Parallel()

	seen := make(map[Capability]struct{}, len(capabilityOrder))
	for _, capability := range capabilityOrder {
		if capability == "" {
			t.Fatal("expected no empty capability key")
		}
		if _, duplicate := seen[capability]; duplicate {
			t.Fatalf("capability %q is declared twice", capability)
		}
		seen[capability] = struct{}{}
	}
}

// TestRoleMatrixUsesDeclaredRolesAndCapabilities keeps the two role declarations in
// sync: every grant must reference a declared role and capability, and every declared
// role must appear in the matrix (otherwise it silently receives no capabilities).
func TestRoleMatrixUsesDeclaredRolesAndCapabilities(t *testing.T) {
	t.Parallel()

	declaredRoles := make(map[string]struct{}, len(identity.Roles()))
	for _, role := range identity.Roles() {
		declaredRoles[role.Key] = struct{}{}
	}

	declaredCapabilities := make(map[Capability]struct{}, len(capabilityOrder))
	for _, capability := range capabilityOrder {
		declaredCapabilities[capability] = struct{}{}
	}

	for role := range declaredRoles {
		if _, granted := roleCapabilities[role]; !granted {
			t.Fatalf("role %q is declared in identity.Roles() but has no matrix entry", role)
		}
	}

	for role, capabilities := range roleCapabilities {
		if _, known := declaredRoles[role]; !known {
			t.Fatalf("role %q is not declared in identity.Roles()", role)
		}

		seen := make(map[Capability]struct{}, len(capabilities))
		for _, capability := range capabilities {
			if _, known := declaredCapabilities[capability]; !known {
				t.Fatalf("role %q grants undeclared capability %q", role, capability)
			}
			if _, duplicate := seen[capability]; duplicate {
				t.Fatalf("role %q grants capability %q twice", role, capability)
			}
			seen[capability] = struct{}{}
		}
	}
}

// TestCapabilityCatalogIsFullyGranted asserts that every declared capability is
// reachable: either from a role, or from request-time policy.
func TestCapabilityCatalogIsFullyGranted(t *testing.T) {
	t.Parallel()

	reachable := make(map[Capability]struct{}, len(capabilityOrder))
	for _, capabilities := range roleCapabilities {
		for _, capability := range capabilities {
			reachable[capability] = struct{}{}
		}
	}
	for _, capability := range expectedPolicyDerived {
		reachable[capability] = struct{}{}
	}

	for _, capability := range capabilityOrder {
		if _, ok := reachable[capability]; !ok {
			t.Fatalf("capability %q is declared but never granted by a role or by policy", capability)
		}
	}
}

func TestPolicyDerivedCapabilitiesAreNotGrantedByRoles(t *testing.T) {
	t.Parallel()

	for _, derived := range expectedPolicyDerived {
		for role, capabilities := range roleCapabilities {
			if slices.Contains(capabilities, derived) {
				t.Fatalf("policy-derived capability %q must not be granted to role %q", derived, role)
			}
		}
	}
}

func TestAllReturnsIndependentCopy(t *testing.T) {
	t.Parallel()

	first := All()
	if len(first) == 0 {
		t.Fatal("expected a non-empty capability catalog")
	}

	first[0] = "mutated"
	if second := All(); second[0] == "mutated" {
		t.Fatal("expected All to return a defensive copy")
	}
}
