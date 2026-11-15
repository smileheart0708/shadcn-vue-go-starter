<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ServerCrash } from '@lucide/vue'
import { Button } from '@/components/ui/button'

const { t } = useI18n()

// A full reload is the only honest retry: the call that failed runs before the router exists
// and before any page is mounted, so there is no smaller unit to re-run.
function retry() {
  window.location.reload()
}
</script>

<template>
  <main class="flex items-center justify-center p-6 min-block-svh">
    <div class="flex flex-col items-center gap-6 text-center max-inline-sm">
      <div
        class="flex items-center justify-center rounded-full bg-muted text-muted-foreground block-16 inline-16"
      >
        <ServerCrash class="block-7 inline-7" />
      </div>

      <h1 class="text-lg font-semibold">
        {{ t('common.feedback.serviceUnavailable') }}
      </h1>

      <Button
        size="lg"
        @click="retry"
      >
        {{ t('common.action.retry') }}
      </Button>
    </div>
  </main>
</template>
