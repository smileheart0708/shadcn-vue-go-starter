import type { RouteLocationNormalizedLoaded } from 'vue-router'
import type { MessageSchema } from '@/locales'
import i18n from '@/plugins/i18n'

// The `route` block is flat (every leaf is a string), so its top-level keys are
// the valid title keys.
export type RouteTitleKey =
  `route.${Extract<keyof MessageSchema['route'], string>}`

export function getRouteTitleKey(
  route: Pick<RouteLocationNormalizedLoaded, 'matched'>,
): RouteTitleKey | null {
  for (const record of [...route.matched].reverse()) {
    if (record.meta.titleKey) {
      return record.meta.titleKey
    }
  }

  return null
}

/**
 * Resolves a route title key to localized copy.
 *
 * The switch must stay exhaustive and every branch must call `t()` with a literal:
 * `@intlify/vue-i18n/no-unused-keys` only sees literal keys inside `t()` calls, so
 * a lookup table here would make the whole `route.*` block look unused.
 */
export function translateRouteTitle(key: RouteTitleKey): string {
  switch (key) {
    case 'route.systemConfig':
      return i18n.global.t('route.systemConfig')
    case 'route.adminUsers':
      return i18n.global.t('route.adminUsers')
    case 'route.dashboard':
      return i18n.global.t('route.dashboard')
    case 'route.login':
      return i18n.global.t('route.login')
    case 'route.notFound':
      return i18n.global.t('route.notFound')
    case 'route.settings':
      return i18n.global.t('route.settings')
    case 'route.register':
      return i18n.global.t('route.register')
    case 'route.setup':
      return i18n.global.t('route.setup')
    case 'route.systemLogs':
      return i18n.global.t('route.systemLogs')
    default:
      return assertNever(key)
  }
}

function assertNever(value: never): never {
  throw new Error(`untranslated route title key: ${String(value)}`)
}
