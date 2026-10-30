import 'vue-router'
import type { Capability } from '@/lib/auth/capabilities.generated'
import type { RouteTitleKey } from '@/router/route-title'

declare module 'vue-router' {
  interface RouteMeta {
    titleKey?: RouteTitleKey
    requiresAuth?: boolean
    guestOnly?: boolean
    requiredCapabilities?: Capability[]
    maskUnauthorizedAsNotFound?: boolean
  }
}

export {}
