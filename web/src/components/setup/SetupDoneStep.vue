<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import type { SetupWizard } from '@/composables/setup/types'

const { wizard } = defineProps<{ wizard: SetupWizard }>()
const { t } = useI18n()
const autoLoginFailed = computed(() => wizard.autoLoginFailed.value)
</script>

<template>
  <div
    class="flex flex-1 flex-col items-center justify-center gap-8 py-6 text-center sm:py-10"
  >
    <div
      class="flex items-center justify-center rounded-full bg-primary/10 text-primary block-20 inline-20"
    >
      <Check class="block-10 inline-10" />
    </div>

    <div class="space-y-3">
      <h2 class="text-3xl font-bold tracking-tight">
        {{ t('setup.done.title') }}
      </h2>
      <p class="text-sm text-muted-foreground">
        {{
          autoLoginFailed
            ? t('setup.done.autoLoginFailed')
            : t('setup.done.description')
        }}
      </p>
    </div>

    <Button
      size="lg"
      class="px-8 inline-full sm:inline-auto"
      @click="wizard.finish()"
    >
      {{
        autoLoginFailed ? t('setup.done.goLogin') : t('setup.done.goDashboard')
      }}
    </Button>
  </div>
</template>
