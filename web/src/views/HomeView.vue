<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowRightIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'

// Marketing landing page shipped with the starter — replace the copy with your own.
const { t } = useI18n()
</script>

<template>
  <main
    class="flex items-center justify-center bg-background text-foreground min-block-screen"
  >
    <div class="flex flex-col items-center px-6 text-center max-inline-5xl">
      <a
        href="#"
        class="inline-flex items-center gap-2 rounded-full border border-border bg-card px-4 py-2 text-sm font-medium transition-colors hover:bg-accent"
      >
        <span class="rounded-full bg-[#22c55e] block-2 inline-2" />
        <span>{{ t('home.announcement') }}</span>
        <ArrowRightIcon class="block-4 inline-4" />
      </a>

      <h1
        class="mbs-4 text-[2rem] font-semibold tracking-[-0.04em] text-balance text-foreground max-inline-4xl sm:text-[2.5rem] lg:text-[3rem]"
      >
        {{ t('home.title') }}
      </h1>

      <p
        class="mbs-3 text-sm/relaxed text-balance text-muted-foreground max-inline-3xl sm:text-base/relaxed"
      >
        {{ t('home.description') }}
      </p>

      <div class="mbs-5 flex flex-wrap items-center justify-center gap-4">
        <Button
          as-child
          size="lg"
          class="px-5 text-sm block-10"
        >
          <RouterLink to="/dashboard">{{ t('home.getStarted') }}</RouterLink>
        </Button>
        <Button
          as-child
          variant="ghost"
          size="lg"
          class="px-4 text-sm block-10"
        >
          <a href="#components">{{ t('home.viewComponents') }}</a>
        </Button>
      </div>
    </div>
  </main>
</template>
