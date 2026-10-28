import type { Capability, RoleKey } from '@/lib/api/auth'

const ROLE_KEY = {
  owner: 'owner',
  user: 'user',
} as const satisfies Record<string, RoleKey>

/**
 * Named handles for the capability strings the backend grants in `viewer.authorization`.
 * Always reference capabilities through this object — a bare string literal type-checks
 * (the type is a string-literal union) but silently drifts if the backend renames one.
 */
export const CAPABILITY = {
  systemSettingsRead: 'system.settings.read',
  systemSettingsUpdate: 'system.settings.update',
  managementUsersRead: 'management.users.read',
  managementUsersCreate: 'management.users.create',
  managementUsersUpdate: 'management.users.update',
  managementUsersDisable: 'management.users.disable',
  managementUsersEnable: 'management.users.enable',
  managementSystemLogsRead: 'management.system_logs.read',
  accountDeleteSelf: 'account.delete_self',
} as const satisfies Record<string, Capability>

export function getUserRoleLabelKey(role: RoleKey | null | undefined) {
  switch (role) {
    case ROLE_KEY.owner:
      return 'common.role.owner'
    default:
      return 'common.role.user'
  }
}

export function getUserRoleBadgeVariant(
  role: RoleKey | null | undefined,
): 'warning' | 'outline' {
  switch (role) {
    case ROLE_KEY.owner:
      return 'warning'
    default:
      return 'outline'
  }
}
