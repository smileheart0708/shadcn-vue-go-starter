<script setup lang="ts">
const props = defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <div
    class="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-8"
  >
    <div class="space-y-1">
      <p class="text-sm font-medium">{{ props.title }}</p>
      <p
        v-if="props.description !== undefined"
        class="text-sm text-muted-foreground"
      >
        {{ props.description }}
      </p>
    </div>

    <div class="flex items-center gap-2 inline-full sm:inline-auto">
      <slot />
    </div>
  </div>
</template>
