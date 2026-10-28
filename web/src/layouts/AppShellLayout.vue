<script setup lang="ts">
import AppShellHeader from '@/components/app-shell/AppShellHeader.vue'
import AppShellSidebar from '@/components/app-shell/AppShellSidebar.vue'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { useCurrentViewerPolling } from '@/composables/auth/useCurrentViewerPolling'
import { RouterView } from 'vue-router'

// Runs for the whole authenticated area: a server-side change to the viewer's status or
// capabilities should take effect on any page, not only while Settings is open.
useCurrentViewerPolling()
</script>

<template>
  <SidebarProvider class="bg-background">
    <AppShellSidebar />
    <SidebarInset>
      <AppShellHeader />
      <div class="flex flex-1 flex-col min-block-0">
        <RouterView />
      </div>
    </SidebarInset>
  </SidebarProvider>
</template>
