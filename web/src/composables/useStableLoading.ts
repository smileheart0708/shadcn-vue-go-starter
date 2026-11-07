import { computed, onScopeDispose, readonly, ref, watch, type Ref } from 'vue'

/**
 * Same perceived-latency policy as the top route loading bar (`router/route-loading.ts`):
 * appear only after 120ms, stay at least 80ms. The two keep their own numbers rather than
 * sharing a module — one is a global singleton with a request counter, the other a per-page
 * scope switch — but they are tuned as a pair, so change both.
 */
export const LOADING_SHOW_DELAY_MS = 120
export const LOADING_MIN_VISIBLE_MS = 80

/**
 * Route changes remount page composables, which start with `loading=true`. A response that
 * arrives in a few milliseconds then replaced a freshly painted skeleton, so navigation
 * looked like a one-frame flash. Same policy as the top route loading bar: wait before
 * showing, and never hide immediately after showing.
 *
 * Two outputs because the suppressed window needs a third rendering:
 * - `showLoading` paints the skeleton.
 * - `firstLoadPending` means the first response is still in flight while the skeleton is
 *   deliberately withheld. Consumers must render neither the skeleton nor their empty
 *   state during that window, or the suppression turns into an empty-state flash.
 */
export function useStableLoading(pending: Ref<boolean>) {
  const showLoading = ref(false)
  // Set once any request settles: a later reload has content on screen worth keeping, so it
  // must not fall back into the first-paint window.
  const hasResolved = ref(false)
  let showTimer: ReturnType<typeof setTimeout> | null = null
  let hideTimer: ReturnType<typeof setTimeout> | null = null
  let visibleSince = 0

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

  watch(
    pending,
    (isPending) => {
      if (isPending) {
        clearHideTimer()
        if (showLoading.value || showTimer !== null) {
          return
        }

        showTimer = setTimeout(() => {
          showTimer = null
          showLoading.value = true
          visibleSince = Date.now()
        }, LOADING_SHOW_DELAY_MS)
        return
      }

      hasResolved.value = true
      clearShowTimer()
      if (!showLoading.value) {
        return
      }

      const remainingMs = Math.max(
        0,
        LOADING_MIN_VISIBLE_MS - (Date.now() - visibleSince),
      )
      hideTimer = setTimeout(() => {
        hideTimer = null
        showLoading.value = false
      }, remainingMs)
    },
    { immediate: true },
  )

  const firstLoadPending = computed(
    () => pending.value && !showLoading.value && !hasResolved.value,
  )

  onScopeDispose(() => {
    clearShowTimer()
    clearHideTimer()
  })

  return {
    showLoading: readonly(showLoading),
    firstLoadPending,
  }
}
