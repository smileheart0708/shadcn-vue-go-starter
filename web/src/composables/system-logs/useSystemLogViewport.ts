import { useElementSize } from '@vueuse/core'
import { nextTick, ref, useTemplateRef } from 'vue'

const SCROLL_BOTTOM_THRESHOLD = 24

export function useSystemLogViewport() {
  const viewport = useTemplateRef<HTMLDivElement>('viewport')
  const { height: viewportHeight } = useElementSize(viewport)
  const scrollTop = ref(0)
  const autoScroll = ref(true)

  function pauseAutoScroll() {
    autoScroll.value = false
  }

  function resumeAutoScroll() {
    autoScroll.value = true
    void scrollToBottom()
  }

  function toggleAutoScroll() {
    if (autoScroll.value) {
      pauseAutoScroll()
      return
    }

    resumeAutoScroll()
  }

  async function scrollToBottom() {
    await nextTick()
    if (!viewport.value) {
      return
    }
    viewport.value.scrollTop = viewport.value.scrollHeight
    // The scroll event that usually refreshes this fires a frame later; updating it
    // here lets the virtual window render the new tail before the next paint.
    scrollTop.value = viewport.value.scrollTop
  }

  function handleViewportScroll(event: Event) {
    const target = event.target
    if (!(target instanceof HTMLDivElement)) {
      return
    }

    scrollTop.value = target.scrollTop
    const distanceFromBottom =
      target.scrollHeight - target.scrollTop - target.clientHeight
    autoScroll.value = distanceFromBottom <= SCROLL_BOTTOM_THRESHOLD
  }

  // resumeAutoScroll stays private: only toggleAutoScroll drives it today, and
  // exporting it invited callers to bypass the toggle's scroll position handling.
  return {
    autoScroll,
    scrollTop,
    viewportHeight,
    toggleAutoScroll,
    scrollToBottom,
    handleViewportScroll,
  }
}
