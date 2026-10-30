import type { RouteLocationNormalized, Router } from 'vue-router'
import { hasAllCapabilities } from '@/lib/auth/authorize'
import pinia from '@/stores/pinia'
import { useAuthStore } from '@/stores/auth'

function resolveNotFoundLocation(
  route: Pick<RouteLocationNormalized, 'path' | 'query' | 'hash'>,
) {
  return {
    name: 'not-found' as const,
    params: {
      pathMatch: route.path.split('/').filter(Boolean),
    },
    query: route.query,
    hash: route.hash || undefined,
  }
}

export function installAuthGuard(router: Router) {
  router.beforeEach(async (to) => {
    const authStore = useAuthStore(pinia)
    const requiresAuth = to.matched.some(
      (record) => record.meta.requiresAuth === true,
    )
    const guestOnly = to.matched.some(
      (record) => record.meta.guestOnly === true,
    )
    const maskUnauthorizedAsNotFound = to.matched.some(
      (record) => record.meta.maskUnauthorizedAsNotFound === true,
    )
    const requiredCapabilities = to.matched.flatMap(
      (record) => record.meta.requiredCapabilities ?? [],
    )

    await authStore.initialize()

    if (!authStore.isSetupComplete && to.name !== 'setup') {
      return { name: 'setup' }
    }

    if (authStore.isSetupComplete && to.name === 'setup') {
      return authStore.isAuthenticated
        ? { name: 'dashboard' }
        : { name: 'login' }
    }

    if (guestOnly && authStore.isAuthenticated) {
      return { name: 'dashboard' }
    }

    if (requiresAuth && !authStore.isAuthenticated) {
      return {
        name: 'login',
        query: to.fullPath === '/login' ? undefined : { redirect: to.fullPath },
      }
    }

    if (
      requiredCapabilities.length > 0 &&
      !hasAllCapabilities(authStore.capabilities, requiredCapabilities)
    ) {
      if (maskUnauthorizedAsNotFound) {
        return resolveNotFoundLocation(to)
      }

      return authStore.isAuthenticated
        ? { name: 'dashboard' }
        : { name: 'login' }
    }

    return true
  })
}
