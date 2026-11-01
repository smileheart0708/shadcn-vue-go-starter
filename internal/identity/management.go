package identity

import "slices"

// ManagedUserAction is an action an operator can perform on another account.
type ManagedUserAction string

const (
	ManagedUserActionUpdate  ManagedUserAction = "update"
	ManagedUserActionDisable ManagedUserAction = "disable"
	ManagedUserActionEnable  ManagedUserAction = "enable"
)

// managedUserActionOrder is the authoritative list of every action the API carries, in
// the order the generated frontend enum follows.
//
// Keep it in sync with the const block above: Go cannot reflect constants, so an action
// missing from this list still compiles and passes every test while the frontend never
// learns about it.
var managedUserActionOrder = []ManagedUserAction{
	ManagedUserActionUpdate,
	ManagedUserActionDisable,
	ManagedUserActionEnable,
}

// AllManagedUserActions returns every declared action in declaration order.
//
// Exported for the frontend contract generator; no production code calls it. The
// per-target subset is ManagedUserActions, which is a rule and not a catalog.
func AllManagedUserActions() []ManagedUserAction {
	return slices.Clone(managedUserActionOrder)
}

// ManagedUserTarget is the minimum a rule needs to know about an actor or a
// target account. Status is only read to decide which status toggle applies.
type ManagedUserTarget struct {
	UserID int64
	Role   string
	Status string
}

// CanManageUser reports whether actor may perform action on target.
//
// It encodes object-level relationships only: an operator can never act on
// itself and never on a peer or higher-ranked account. Whether the actor may
// perform this class of action at all is decided by
// httpapi.RequireCapability, so the two checks stay disjoint. This is why the
// rule compares ranks instead of role names: a role granted the capability is
// automatically allowed down to its own rank.
func CanManageUser(actor ManagedUserTarget, target ManagedUserTarget, action ManagedUserAction) bool {
	if actor.UserID == target.UserID {
		return false
	}

	actorRank, actorKnown := RoleRank(actor.Role)
	if !actorKnown {
		return false
	}
	targetRank, targetKnown := RoleRank(target.Role)
	if !targetKnown {
		return false
	}
	if actorRank <= targetRank {
		return false
	}

	switch action {
	case ManagedUserActionUpdate, ManagedUserActionDisable, ManagedUserActionEnable:
		return true
	default:
		return false
	}
}

// ManagedUserActions lists the actions available against target in its current
// status. It is the same rule as CanManageUser, narrowed by status, so the UI
// cannot offer an action the API would reject.
func ManagedUserActions(actor ManagedUserTarget, target ManagedUserTarget) []ManagedUserAction {
	actions := make([]ManagedUserAction, 0, 2)
	if CanManageUser(actor, target, ManagedUserActionUpdate) {
		actions = append(actions, ManagedUserActionUpdate)
	}

	switch target.Status {
	case StatusActive:
		if CanManageUser(actor, target, ManagedUserActionDisable) {
			actions = append(actions, ManagedUserActionDisable)
		}
	case StatusDisabled:
		if CanManageUser(actor, target, ManagedUserActionEnable) {
			actions = append(actions, ManagedUserActionEnable)
		}
	}

	return actions
}
