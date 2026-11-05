<script setup lang="ts">
import { useEventListener } from '@vueuse/core'
import { nextTick, useTemplateRef, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppShellHeader from '@/components/app-shell/AppShellHeader.vue'
import AppShellSidebar from '@/components/app-shell/AppShellSidebar.vue'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { useCurrentViewerPolling } from '@/composables/auth/useCurrentViewerPolling'

const route = useRoute()

// Runs for the whole authenticated area: a server-side change to the viewer's status or
// capabilities should take effect on any page, not only while Settings is open.
useCurrentViewerPolling()

const contentView = useTemplateRef<HTMLElement>('contentView')

// The content view is the only scroll container, so the browser no longer restores a
// history entry's offset for us: a fresh navigation has to start at the top, and a
// back/forward has to return to where that entry was left. `popstate` is the only
// signal that tells the two apart — vue-router's own `savedPosition` tracks `window`,
// which never scrolls here.
const scrollPositions = new Map<string, number>()
let isHistoryNavigation = false

useEventListener('popstate', () => {
  isHistoryNavigation = true
})

watch(
  () => route.fullPath,
  async (to, from) => {
    // Consumed before anything can return early: a leaked `true` would turn the next
    // ordinary navigation into a restore.
    const restorePosition = isHistoryNavigation
    isHistoryNavigation = false

    const view = contentView.value
    if (view === null) {
      return
    }

    // Default `pre` flush: the outgoing page is still mounted, so this reads its offset.
    scrollPositions.set(from, view.scrollTop)

    await nextTick()
    // The restored offset is clamped by the height the target page has after its first
    // render; a page whose content arrives later simply comes back higher.
    view.scrollTo({ top: restorePosition ? (scrollPositions.get(to) ?? 0) : 0 })
  },
)
</script>

<template>
  <SidebarProvider class="bg-background">
    <AppShellSidebar />
    <SidebarInset class="overflow-hidden">
      <AppShellHeader />
      <div
        ref="contentView"
        class="flex flex-1 flex-col overflow-y-auto min-block-0"
      >
        <RouterView />
      </div>
    </SidebarInset>
  </SidebarProvider>
</template>
