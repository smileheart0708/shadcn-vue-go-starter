<script setup lang="ts">
import type { AppShellNavItem } from '@/components/app-shell/navigation'
import AppShellNavButton from '@/components/app-shell/AppShellNavButton.vue'
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuItem,
} from '@/components/ui/sidebar'

defineProps<{
  items: AppShellNavItem[]
  label?: string
}>()
</script>

<template>
  <!-- An empty group still paints SidebarGroup's padding, so skip it entirely. -->
  <SidebarGroup v-if="items.length > 0">
    <SidebarGroupLabel v-if="label">{{ label }}</SidebarGroupLabel>
    <SidebarGroupContent>
      <SidebarMenu>
        <SidebarMenuItem
          v-for="item in items"
          :key="item.title"
        >
          <!--
            No tooltip here: `AppShellSidebar` only supports
            `collapsible="offcanvas"`, where `SidebarMenuButton` keeps its
            tooltip hidden (`hidden` is tied to `state === 'collapsed'`) and the
            collapsed panel is translated off-screen, so the content could never
            be read. Add `:tooltip` back together with an icon-collapse mode.
          -->
          <AppShellNavButton :item="item" />
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarGroupContent>
  </SidebarGroup>
</template>
