<script setup lang="ts">
import type { SidebarProps } from '.'
import { cn } from '@/lib/utils'
import { Sheet, SheetContent } from '@/components/ui/sheet'
import SheetDescription from '@/components/ui/sheet/SheetDescription.vue'
import SheetHeader from '@/components/ui/sheet/SheetHeader.vue'
import SheetTitle from '@/components/ui/sheet/SheetTitle.vue'
import { SIDEBAR_WIDTH_SHEET, useSidebar } from './utils'

const props = withDefaults(defineProps<SidebarProps>(), {
  side: 'left',
  variant: 'sidebar',
  collapsible: 'offcanvas',
})

defineOptions({ inheritAttrs: false })

const { isNarrow, state, openSheet, setOpenSheet } = useSidebar()
</script>

<template>
  <div
    v-if="collapsible === 'none'"
    data-slot="sidebar"
    :class="
      cn(
        'flex flex-col bg-sidebar text-sidebar-foreground block-full inline-(--sidebar-width)',
        props.class,
      )
    "
    v-bind="$attrs"
  >
    <slot />
  </div>

  <!--
    Narrow layout: the sidebar is an overlay sheet. The wide branch below is gated by the
    `wide:` variant, which is the same query `isNarrow` negates, so the two never overlap.
  -->
  <Sheet
    v-else-if="isNarrow"
    :open="openSheet"
    v-bind="$attrs"
    @update:open="setOpenSheet"
  >
    <SheetContent
      data-sidebar="sidebar"
      data-slot="sidebar"
      data-narrow="true"
      :side="side"
      class="bg-sidebar p-0 text-sidebar-foreground inline-(--sidebar-width) [&>button]:hidden"
      :style="{ '--sidebar-width': SIDEBAR_WIDTH_SHEET }"
    >
      <SheetHeader class="sr-only">
        <SheetTitle>Sidebar</SheetTitle>
        <SheetDescription>Displays the sidebar as a sheet.</SheetDescription>
      </SheetHeader>
      <div class="flex flex-col block-full inline-full">
        <slot />
      </div>
    </SheetContent>
  </Sheet>

  <div
    v-else
    class="group peer hidden text-sidebar-foreground wide:block"
    data-slot="sidebar"
    :data-state="state"
    :data-collapsible="state === 'collapsed' ? collapsible : ''"
    :data-variant="variant"
    :data-side="side"
  >
    <!-- The gap the fixed wide bar occupies; the narrow layout has no gap. -->
    <div
      :class="
        cn(
          'relative bg-transparent transition-[width] duration-200 ease-linear inline-(--sidebar-width)',
          'group-data-[collapsible=offcanvas]:inline-0',
          'group-data-[side=right]:rotate-180',
          variant === 'floating' || variant === 'inset'
            ? 'group-data-[collapsible=icon]:inline-[calc(var(--sidebar-width-icon)+(--spacing(4)))]'
            : 'group-data-[collapsible=icon]:inline-(--sidebar-width-icon)',
        )
      "
    />
    <div
      :class="
        cn(
          'fixed inset-y-0 z-10 hidden transition-[left,right,width] duration-200 ease-linear block-svh inline-(--sidebar-width) wide:flex',
          side === 'left'
            ? 'inset-s-0 group-data-[collapsible=offcanvas]:-inset-s-(--sidebar-width)'
            : 'inset-e-0 group-data-[collapsible=offcanvas]:-inset-e-(--sidebar-width)',
          // Adjust the padding for floating and inset variants.
          variant === 'floating' || variant === 'inset'
            ? 'p-2 group-data-[collapsible=icon]:inline-[calc(var(--sidebar-width-icon)+(--spacing(4))+2px)]'
            : 'group-data-[collapsible=icon]:inline-(--sidebar-width-icon) group-data-[side=left]:border-e group-data-[side=right]:border-s',
          props.class,
        )
      "
      v-bind="$attrs"
    >
      <div
        data-sidebar="sidebar"
        class="flex flex-col bg-sidebar block-full inline-full group-data-[variant=floating]:rounded-lg group-data-[variant=floating]:border group-data-[variant=floating]:border-sidebar-border group-data-[variant=floating]:shadow-sm"
      >
        <slot />
      </div>
    </div>
  </div>
</template>
