package authorization

import "main/internal/identity"

// roleCapabilities is the only place a role receives capabilities.
//
// Adding a role means two edits: one entry in identity's role table and one
// entry here. Anything that is not a plain role grant (policy-driven abilities,
// object-level relationships) belongs elsewhere on purpose.
var roleCapabilities = map[string][]Capability{
	identity.RoleOwner: {
		CapabilitySystemSettingsRead,
		CapabilitySystemSettingsUpdate,
		CapabilityManagementUsersRead,
		CapabilityManagementUsersCreate,
		CapabilityManagementUsersUpdate,
		CapabilityManagementUsersDisable,
		CapabilityManagementUsersEnable,
		CapabilityManagementSystemLogsRead,
	},
	identity.RoleUser: {},
}
