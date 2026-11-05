<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'
import { useSidebar } from './utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()

const { toggleSidebar } = useSidebar()
</script>

<template>
  <button
    data-sidebar="rail"
    data-slot="sidebar-rail"
    type="button"
    aria-label="Toggle Sidebar"
    :tabindex="-1"
    title="Toggle Sidebar"
    :class="
      cn(
        'absolute inset-y-0 z-20 hidden -translate-x-1/2 transition-all ease-linear inline-4 group-data-[side=left]:-inset-e-4 group-data-[side=right]:inset-s-0 after:absolute after:inset-y-0 after:inset-s-1/2 after:inline-0.5 hover:after:bg-sidebar-border wide:flex',
        'in-data-[side=left]:cursor-w-resize in-data-[side=right]:cursor-e-resize',
        '[[data-side=left][data-state=collapsed]_&]:cursor-e-resize [[data-side=right][data-state=collapsed]_&]:cursor-w-resize',
        'group-data-[collapsible=offcanvas]:translate-x-0 group-data-[collapsible=offcanvas]:after:inset-s-full hover:group-data-[collapsible=offcanvas]:bg-sidebar',
        '[[data-side=left][data-collapsible=offcanvas]_&]:-inset-e-2',
        '[[data-side=right][data-collapsible=offcanvas]_&]:-inset-s-2',
        props.class,
      )
    "
    @click="toggleSidebar"
  >
    <slot />
  </button>
</template>
