<script setup lang="ts">
import { RouterView } from 'vue-router'
import RouteLoadingBar from '@/components/app/RouteLoadingBar.vue'
import RouteLoadingErrorToast from '@/components/app/RouteLoadingErrorToast.vue'
import { Toaster } from '@/components/ui/sonner'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useThemeStore } from '@/stores/theme'

// Keep the theme store alive at the app root: it owns the system-theme listener and
// persisted preference, which must react on pages without a theme toggle.
useThemeStore()
</script>

<template>
  <TooltipProvider>
    <RouteLoadingBar />
    <RouteLoadingErrorToast />
    <RouterView />
    <Toaster />
  </TooltipProvider>
</template>
