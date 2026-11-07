import { nextTick } from 'vue'
import type { Router } from 'vue-router'
import { THEME_REVEAL_CSS } from '@/stores/theme'

type NavigationTarget = Parameters<Router['push']>[0]
type NavigationResult = Awaited<ReturnType<Router['push']>>

/**
 * Best-effort route cross-fade. Wrapping push/replace (what RouterLink calls) is what lets
 * the old snapshot be captured before the DOM changes: startViewTransition captures at the
 * next rendering opportunity, so a router.afterEach hook would already be too late.
 * Browser back/forward goes through popstate instead and keeps the instant navigation.
 *
 * The instance-level override works because vue-router 5's `useLink` reads
 * `router[replace ? 'replace' : 'push']` on every click. That is an internal detail: if a
 * future version calls the method directly or destructures it, the cross-fade silently
 * degrades to instant navigation. vue-router also ships a per-link `viewTransition` prop,
 * which covers links only — switching to it means adding the prop to every RouterLink.
 */
let installed = false

export function installRouteViewTransitions(router: Router) {
  // Same one-shot contract as installRouteLoading: a second install would wrap both
  // methods twice and nest a transition inside a transition.
  if (installed || !('startViewTransition' in document)) {
    return
  }

  installed = true

  const push = router.push.bind(router)
  const replace = router.replace.bind(router)
  let enabled = false
  router.afterEach(() => {
    enabled = true
  })

  function shouldSkipTransition(to: NavigationTarget) {
    // A hidden tab has no rendering opportunity, so the update callback would never run
    // and the navigation would hang inside the transition.
    return (
      !enabled ||
      document.visibilityState !== 'visible' ||
      router.resolve(to).fullPath === router.currentRoute.value.fullPath
    )
  }

  async function withCrossFade(navigate: () => Promise<NavigationResult>) {
    // theme.ts leaves this switch on <html> until its next theme change; without removing
    // it, navigation would play the theme's circular reveal instead of a cross-fade.
    document.documentElement.removeAttribute(THEME_REVEAL_CSS.attribute)

    let result: NavigationResult = undefined
    const transition = document.startViewTransition(async () => {
      result = await navigate()
      await nextTick()
    })

    await transition.updateCallbackDone
    return result
  }

  router.push = async (to) =>
    shouldSkipTransition(to) ? push(to) : withCrossFade(async () => push(to))
  router.replace = async (to) =>
    shouldSkipTransition(to)
      ? replace(to)
      : withCrossFade(async () => replace(to))
}
