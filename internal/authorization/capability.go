package authorization

import "slices"

// Capability is a named permission unit. Guards and policies reference these
// constants instead of raw strings, so a typo is a compile error rather than a
// silently disabled check.
type Capability string

const (
	CapabilitySystemSettingsRead       Capability = "system.settings.read"
	CapabilitySystemSettingsUpdate     Capability = "system.settings.update"
	CapabilityManagementUsersRead      Capability = "management.users.read"
	CapabilityManagementUsersCreate    Capability = "management.users.create"
	CapabilityManagementUsersUpdate    Capability = "management.users.update"
	CapabilityManagementUsersDisable   Capability = "management.users.disable"
	CapabilityManagementUsersEnable    Capability = "management.users.enable"
	CapabilityManagementSystemLogsRead Capability = "management.system_logs.read"
	CapabilityAccountDeleteSelf        Capability = "account.delete_self"
)

// capabilityOrder is the authoritative list of every declared capability.
//
// The order is part of the contract: the frontend module is generated from it,
// so append new capabilities instead of reordering, otherwise the generated
// diff stops being reviewable.
//
// Keep it in sync with the const block above: Go cannot reflect constants, so a
// const missing from this list still compiles and passes every test while the
// capability silently never reaches the frontend.
var capabilityOrder = []Capability{
	CapabilitySystemSettingsRead,
	CapabilitySystemSettingsUpdate,
	CapabilityManagementUsersRead,
	CapabilityManagementUsersCreate,
	CapabilityManagementUsersUpdate,
	CapabilityManagementUsersDisable,
	CapabilityManagementUsersEnable,
	CapabilityManagementSystemLogsRead,
	CapabilityAccountDeleteSelf,
}

// All returns every declared capability in declaration order.
func All() []Capability {
	return slices.Clone(capabilityOrder)
}
