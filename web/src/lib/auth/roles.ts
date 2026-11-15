import { ROLE_KEY, type RoleKey } from '@/lib/api/contract.generated'
import { assertNever } from '@/utils/assert-never'

type RoleLabelTranslate = (
  key: 'common.role.owner' | 'common.role.user',
) => string

/**
 * Resolves a role label.
 *
 * The `t()` calls must stay literal and inline: `@intlify/vue-i18n/no-unused-keys`
 * only recognises literal keys inside `t()`, so returning a key for the caller to
 * translate would make `common.role.*` look unused.
 *
 * `null`/`undefined` means the viewer has not loaded yet, not "user", so it is handled before
 * the switch and the default branch can be `assertNever`. A single `default: return user` used
 * to absorb both cases, which meant adding a role to the Go contract rendered it as "User" with
 * no warning: the generated type gained a member, this switch kept compiling, nothing failed.
 */
export function getUserRoleLabel(
  t: RoleLabelTranslate,
  role: RoleKey | null | undefined,
): string {
  if (role === null || role === undefined) {
    return t('common.role.user')
  }

  switch (role) {
    case ROLE_KEY.owner:
      return t('common.role.owner')
    case ROLE_KEY.user:
      return t('common.role.user')
    default:
      return assertNever(role)
  }
}

export function getUserRoleBadgeVariant(
  role: RoleKey | null | undefined,
): 'warning' | 'outline' {
  if (role === null || role === undefined) {
    return 'outline'
  }

  switch (role) {
    case ROLE_KEY.owner:
      return 'warning'
    case ROLE_KEY.user:
      return 'outline'
    default:
      return assertNever(role)
  }
}
