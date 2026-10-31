<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <section class="flex items-start justify-between gap-4">
    <div class="space-y-1">
      <!--
        h2 rather than h1: AppShellHeader already renders the page-level h1 from the
        route title. A second h1 here would give every page two top-level headings.
      -->
      <div class="flex flex-wrap items-center gap-3">
        <h2 class="text-2xl font-semibold">{{ title }}</h2>
        <slot name="meta" />
      </div>
      <p
        v-if="description"
        class="text-sm text-muted-foreground"
      >
        {{ description }}
      </p>
    </div>

    <!--
      `meta` and `actions` are separate slots on purpose: a badge belongs next to the
      title (system logs), a button belongs on the far end (user management). Folding
      both into one slot would move the badge to the edge of the page.
    -->
    <div
      v-if="$slots.actions"
      class="flex shrink-0 items-center gap-2"
    >
      <slot name="actions" />
    </div>
  </section>
</template>
