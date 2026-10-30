import type { RoleKey } from '@/lib/api/auth'

export const ROLE_KEY = {
  owner: 'owner',
  user: 'user',
} as const satisfies Record<string, RoleKey>

type RoleLabelTranslate = (
  key: 'common.role.owner' | 'common.role.user',
) => string

/**
 * Resolves a role label.
 *
 * The `t()` calls must stay literal and inline: `@intlify/vue-i18n/no-unused-keys`
 * only recognises literal keys inside `t()`, so returning a key for the caller to
 * translate would make `common.role.*` look unused.
 */
export function getUserRoleLabel(
  t: RoleLabelTranslate,
  role: RoleKey | null | undefined,
): string {
  switch (role) {
    case ROLE_KEY.owner:
      return t('common.role.owner')
    default:
      return t('common.role.user')
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
