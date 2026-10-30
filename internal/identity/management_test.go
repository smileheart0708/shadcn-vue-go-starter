package identity

import (
	"slices"
	"testing"
)

func TestRoleRankRejectsUnknownRoles(t *testing.T) {
	t.Parallel()

	if _, ok := RoleRank("auditor"); ok {
		t.Fatal("expected unknown role to have no rank")
	}
	if _, ok := RoleRank(""); ok {
		t.Fatal("expected empty role to have no rank")
	}
	if _, ok := RoleRank(RoleOwner); !ok {
		t.Fatal("expected owner role to be ranked")
	}
}

func TestRolesAreOrderedByDescendingRank(t *testing.T) {
	t.Parallel()

	roles := Roles()
	if len(roles) < 2 {
		t.Fatalf("expected at least two roles, got %v", roles)
	}
	for index := 1; index < len(roles); index++ {
		if roles[index-1].Rank <= roles[index].Rank {
			t.Fatalf("expected roles ordered by descending rank, got %v", roles)
		}
	}

	// Roles returns a copy: mutating it must not corrupt the declaration.
	declaredRank := roles[0].Rank
	roles[0].Rank = -1
	if fresh := Roles(); fresh[0].Rank != declaredRank {
		t.Fatalf("expected Roles to return a defensive copy, got %v", fresh)
	}
}

func TestCanManageUserEnforcesRankBoundaries(t *testing.T) {
	t.Parallel()

	owner := ManagedUserTarget{UserID: 1, Role: RoleOwner, Status: StatusActive}
	user := ManagedUserTarget{UserID: 2, Role: RoleUser, Status: StatusActive}
	otherOwner := ManagedUserTarget{UserID: 3, Role: RoleOwner, Status: StatusActive}

	for _, tc := range []struct {
		name   string
		actor  ManagedUserTarget
		target ManagedUserTarget
		action ManagedUserAction
		want   bool
	}{
		{name: "owner manages user", actor: owner, target: user, action: ManagedUserActionUpdate, want: true},
		{name: "owner disables user", actor: owner, target: user, action: ManagedUserActionDisable, want: true},
		{name: "owner cannot manage self", actor: owner, target: owner, action: ManagedUserActionUpdate, want: false},
		{name: "owner cannot manage peer owner", actor: owner, target: otherOwner, action: ManagedUserActionUpdate, want: false},
		{name: "user cannot manage peer user", actor: user, target: ManagedUserTarget{UserID: 4, Role: RoleUser, Status: StatusActive}, action: ManagedUserActionDisable, want: false},
		{name: "user cannot manage owner", actor: user, target: owner, action: ManagedUserActionUpdate, want: false},
		{name: "unknown actor role is rejected", actor: ManagedUserTarget{UserID: 9, Role: "auditor", Status: StatusActive}, target: user, action: ManagedUserActionUpdate, want: false},
		{name: "unknown target role is rejected", actor: owner, target: ManagedUserTarget{UserID: 8, Role: "auditor", Status: StatusActive}, action: ManagedUserActionUpdate, want: false},
		{name: "empty target role is rejected", actor: owner, target: ManagedUserTarget{UserID: 7, Status: StatusActive}, action: ManagedUserActionUpdate, want: false},
		{name: "unknown action is rejected", actor: owner, target: user, action: ManagedUserAction("delete"), want: false},
	} {
		if got := CanManageUser(tc.actor, tc.target, tc.action); got != tc.want {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, got)
		}
	}
}

func TestManagedUserActionsFollowTargetStatus(t *testing.T) {
	t.Parallel()

	owner := ManagedUserTarget{UserID: 1, Role: RoleOwner, Status: StatusActive}

	for _, tc := range []struct {
		name   string
		target ManagedUserTarget
		want   []ManagedUserAction
	}{
		{
			name:   "active user exposes update and disable",
			target: ManagedUserTarget{UserID: 2, Role: RoleUser, Status: StatusActive},
			want:   []ManagedUserAction{ManagedUserActionUpdate, ManagedUserActionDisable},
		},
		{
			name:   "disabled user exposes update and enable",
			target: ManagedUserTarget{UserID: 2, Role: RoleUser, Status: StatusDisabled},
			want:   []ManagedUserAction{ManagedUserActionUpdate, ManagedUserActionEnable},
		},
		{
			name:   "self exposes nothing",
			target: owner,
			want:   []ManagedUserAction{},
		},
		{
			name:   "unknown status exposes only update",
			target: ManagedUserTarget{UserID: 3, Role: RoleUser, Status: "pending"},
			want:   []ManagedUserAction{ManagedUserActionUpdate},
		},
	} {
		got := ManagedUserActions(owner, tc.target)
		if got == nil {
			t.Fatalf("%s: expected a non-nil action slice", tc.name)
		}
		if !slices.Equal(got, tc.want) {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, got)
		}
	}
}
