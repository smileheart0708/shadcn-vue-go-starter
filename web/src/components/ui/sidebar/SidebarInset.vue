<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <main
    data-slot="sidebar-inset"
    :class="
      cn(
        'relative flex flex-1 flex-col bg-background inline-full min-block-0 min-inline-0',
        'wide:peer-data-[variant=inset]:m-2 wide:peer-data-[variant=inset]:ms-0 wide:peer-data-[variant=inset]:rounded-xl wide:peer-data-[variant=inset]:shadow-sm wide:peer-data-[variant=inset]:peer-data-[state=collapsed]:ms-2',
        props.class,
      )
    "
  >
    <slot />
  </main>
</template>
