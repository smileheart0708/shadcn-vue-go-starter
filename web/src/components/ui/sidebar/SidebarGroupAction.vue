<script setup lang="ts">
import type { PrimitiveProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { Primitive } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<
  PrimitiveProps & { class?: HTMLAttributes['class'] }
>()
</script>

<template>
  <Primitive
    data-slot="sidebar-group-action"
    data-sidebar="group-action"
    :as="as"
    :as-child="asChild"
    :class="
      cn(
        'absolute inset-e-3 inset-bs-3.5 flex aspect-square items-center justify-center rounded-md p-0 text-sidebar-foreground ring-sidebar-ring outline-hidden transition-transform inline-5 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 [&>svg]:shrink-0 [&>svg]:block-4 [&>svg]:inline-4',
        'after:absolute after:-inset-2 wide:after:hidden',
        'group-data-[collapsible=icon]:hidden',
        props.class,
      )
    "
  >
    <slot />
  </Primitive>
</template>
