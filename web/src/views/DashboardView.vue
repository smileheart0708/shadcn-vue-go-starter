<script setup lang="ts">
import AppShellPage from '@/components/app-shell/AppShellPage.vue'
import SectionCards from '@/components/dashboard/SectionCards.vue'
</script>

<template>
  <!-- The shell header already shows the route title; the dashboard has no in-page heading. -->
  <AppShellPage
    variant="fill"
    hide-page-header
  >
    <SectionCards />
  </AppShellPage>
</template>
