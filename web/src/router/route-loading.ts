import { readonly, ref } from 'vue'
import type { Router } from 'vue-router'

/**
 * Marks a request as background work so it does not drive the top loading bar.
 *
 * Purely a client-side signal — the backend neither reads nor requires it. It lives here
 * rather than in the API client because suppressing the loading bar is its only purpose,
 * and keeping it here makes the dependency one-way (api/client -> route-loading).
 */
export const BACKGROUND_REQUEST_HEADER = 'X-App-Background-Request'

const SHOW_DELAY_MS = 120
const MIN_VISIBLE_MS = 80

const routeLoadingVisible = ref(false)
const routeNavigationPending = ref(false)
const requestLoadingCount = ref(0)
const routeLoadingErrorRevision = ref(0)
const navigationIds = new WeakMap<object, number>()

let installed = false
let activeNavigationId = 0
let visibleSince = 0
let showTimer: ReturnType<typeof setTimeout> | null = null
let hideTimer: ReturnType<typeof setTimeout> | null = null

function hasPendingWork() {
  return routeNavigationPending.value || requestLoadingCount.value > 0
}

function clearShowTimer() {
  if (showTimer === null) {
    return
  }

  clearTimeout(showTimer)
  showTimer = null
}

function clearHideTimer() {
  if (hideTimer === null) {
    return
  }

  clearTimeout(hideTimer)
  hideTimer = null
}

function showRouteLoadingIfPending() {
  showTimer = null

  if (!hasPendingWork()) {
    return
  }

  routeLoadingVisible.value = true
  visibleSince = Date.now()
}

function hideRouteLoadingIfIdle() {
  hideTimer = null

  if (hasPendingWork()) {
    return
  }

  routeLoadingVisible.value = false
  visibleSince = 0
}

function beginNavigation() {
  activeNavigationId += 1
  const navigationId = activeNavigationId

  routeNavigationPending.value = true
  syncLoadingState()

  return navigationId
}

function finishNavigation(navigationId: number) {
  if (navigationId !== activeNavigationId) {
    return
  }

  routeNavigationPending.value = false
  syncLoadingState()
}

function syncLoadingState() {
  if (hasPendingWork()) {
    clearHideTimer()

    if (routeLoadingVisible.value || showTimer !== null) {
      return
    }

    showTimer = setTimeout(showRouteLoadingIfPending, SHOW_DELAY_MS)
    return
  }

  clearShowTimer()

  if (!routeLoadingVisible.value) {
    return
  }

  const visibleForMs = visibleSince > 0 ? Date.now() - visibleSince : 0
  const remainingVisibleMs = Math.max(0, MIN_VISIBLE_MS - visibleForMs)

  clearHideTimer()
  hideTimer = setTimeout(hideRouteLoadingIfIdle, remainingVisibleMs)
}

function incrementRequestLoading() {
  requestLoadingCount.value += 1
  syncLoadingState()
}

function decrementRequestLoading() {
  requestLoadingCount.value = Math.max(0, requestLoadingCount.value - 1)
  syncLoadingState()
}

function signalRouteLoadingError() {
  routeLoadingErrorRevision.value += 1
}

function shouldTrackRequest(request: Request) {
  // Background work (polling, prefetch) must not flash the top loading bar.
  if (request.headers.has(BACKGROUND_REQUEST_HEADER)) {
    return false
  }

  try {
    const locationOrigin =
      typeof globalThis.location === 'undefined'
        ? 'http://localhost'
        : globalThis.location.origin

    const requestUrl = new URL(request.url)
    return (
      requestUrl.origin === locationOrigin &&
      requestUrl.pathname.startsWith('/api/')
    )
  } catch {
    return false
  }
}

export function installRouteLoading(router: Router) {
  if (installed) {
    return
  }

  installed = true

  router.beforeEach((to) => {
    const navigationId = beginNavigation()
    navigationIds.set(to, navigationId)
    return true
  })

  router.afterEach((to) => {
    const navigationId = navigationIds.get(to)
    if (navigationId === undefined) {
      return
    }

    navigationIds.delete(to)
    finishNavigation(navigationId)
  })

  router.onError((_, to) => {
    const navigationId = navigationIds.get(to)
    if (navigationId === undefined) {
      return
    }

    navigationIds.delete(to)

    if (navigationId === activeNavigationId) {
      finishNavigation(navigationId)
      signalRouteLoadingError()
    }
  })
}

/**
 * `fetch` implementation for the shared ky instances that keeps the top loading bar in
 * sync with in-flight API calls.
 *
 * Scoped to ky on purpose: patching `globalThis.fetch` would also intercept the MSW
 * service worker and any third-party script, and leaves a process-wide side effect that
 * is easy to double-install and impossible to reason about from a call site.
 */
export const trackedFetch: typeof globalThis.fetch = async (input, init) => {
  const request = input instanceof Request ? input : new Request(input, init)

  if (!shouldTrackRequest(request)) {
    return globalThis.fetch(input, init)
  }

  incrementRequestLoading()

  try {
    return await globalThis.fetch(input, init)
  } finally {
    decrementRequestLoading()
  }
}

export function useRouteLoadingState() {
  return readonly(routeLoadingVisible)
}

export function useRouteLoadingErrorRevision() {
  return readonly(routeLoadingErrorRevision)
}
