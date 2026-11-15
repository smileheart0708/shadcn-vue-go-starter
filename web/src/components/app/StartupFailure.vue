<script setup lang="ts">
import { useI18n } from 'vue-i18n'
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
    <div class="flex flex-col items-center gap-4 text-center max-inline-sm">
      <h1 class="text-lg font-semibold">
        {{ t('common.feedback.loadFailed') }}
      </h1>
      <p class="text-sm text-muted-foreground">
        {{ t('common.feedback.networkError') }}
      </p>
      <Button @click="retry">{{ t('common.action.retry') }}</Button>
    </div>
  </main>
</template>
