import { nextTick, ref, useTemplateRef } from 'vue'

const SCROLL_BOTTOM_THRESHOLD = 24

export function useSystemLogViewport() {
  const viewport = useTemplateRef<HTMLDivElement>('viewport')
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
  }

  function handleViewportScroll(event: Event) {
    const target = event.target
    if (!(target instanceof HTMLDivElement)) {
      return
    }

    const distanceFromBottom =
      target.scrollHeight - target.scrollTop - target.clientHeight
    autoScroll.value = distanceFromBottom <= SCROLL_BOTTOM_THRESHOLD
  }

  // resumeAutoScroll stays private: only toggleAutoScroll drives it today, and
  // exporting it invited callers to bypass the toggle's scroll position handling.
  return {
    autoScroll,
    toggleAutoScroll,
    scrollToBottom,
    handleViewportScroll,
  }
}
