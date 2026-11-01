import type { RoleKey } from '@/lib/api/auth'
import { CAPABILITY, type Capability } from '@/lib/auth/capabilities.generated'
import { ROLE_KEY } from '@/lib/auth/roles'
import { state, type MockUser } from '@/mocks/handlers/state'

/**
 * The role matrix below mirrors `internal/authorization/policy.go`;
 * `satisfies Record<RoleKey, ...>` makes a new backend role a typecheck failure here
 * until the mock is taught about it.
 */
const ROLE_CAPABILITIES = {
  [ROLE_KEY.owner]: [
    CAPABILITY.systemSettingsRead,
    CAPABILITY.systemSettingsUpdate,
    CAPABILITY.managementUsersRead,
    CAPABILITY.managementUsersCreate,
    CAPABILITY.managementUsersUpdate,
    CAPABILITY.managementUsersDisable,
    CAPABILITY.managementUsersEnable,
    CAPABILITY.managementSystemLogsRead,
  ],
  [ROLE_KEY.user]: [],
} as const satisfies Record<RoleKey, readonly Capability[]>

function capabilitiesFor(user: MockUser): Capability[] {
  const capabilities = new Set<Capability>(ROLE_CAPABILITIES[user.role])

  // Mirrors the server rule: self-deletion is policy-driven and never granted to
  // the owner, who could otherwise delete the only administrative account.
  if (
    state.accountPolicies.selfServiceAccountDeletionEnabled &&
    user.role !== ROLE_KEY.owner
  ) {
    capabilities.add(CAPABILITY.accountDeleteSelf)
  }

  return [...capabilities].sort()
}

export function hasCapability(user: MockUser, capability: Capability) {
  return capabilitiesFor(user).includes(capability)
}

export function toViewer(user: MockUser) {
  return {
    identity: {
      id: user.id,
      username: user.username,
      email: user.email,
      avatarUrl: user.avatarUrl,
      status: user.status,
    },
    authorization: {
      role: user.role,
      capabilities: capabilitiesFor(user),
    },
  }
}
