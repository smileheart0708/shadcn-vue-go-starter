<script setup lang="ts">
import PageHeader from '@/components/app/PageHeader.vue'
import { useRouteTitle } from '@/composables/useRouteTitle'

// `fill` pins the page inside the shell's scroll viewport so a nested panel can own
// scrolling; `flow` lets the shell scroll the whole page instead.
type AppShellPageVariant = 'flow' | 'fill'

const props = withDefaults(
  defineProps<{
    variant?: AppShellPageVariant
    hidePageHeader?: boolean
  }>(),
  { variant: 'flow' },
)

defineSlots<{
  default: () => unknown
  meta?: () => unknown
  actions?: () => unknown
}>()

const title = useRouteTitle()
</script>

<template>
  <div
    class="flex flex-1 flex-col p-4 lg:p-6"
    :class="
      props.variant === 'fill'
        ? 'gap-4 lg:gap-6 lg:overflow-hidden lg:min-block-0'
        : 'gap-6'
    "
  >
    <!--
      `hidePageHeader` drops `meta`/`actions` too: both slots live inside PageHeader,
      so a page that hides the heading cannot keep the title-row buttons.
    -->
    <PageHeader
      v-if="!props.hidePageHeader"
      :title="title"
    >
      <template
        v-if="$slots.meta"
        #meta
      >
        <slot name="meta" />
      </template>
      <template
        v-if="$slots.actions"
        #actions
      >
        <slot name="actions" />
      </template>
    </PageHeader>

    <slot />
  </div>
</template>
