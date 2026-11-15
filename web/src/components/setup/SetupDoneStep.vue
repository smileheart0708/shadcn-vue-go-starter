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
  <div class="flex flex-col items-center gap-6 py-6 text-center sm:py-10">
    <div
      class="flex items-center justify-center rounded-full bg-primary/10 text-primary block-20 inline-20"
    >
      <Check class="block-10 inline-10" />
    </div>

    <h1 class="text-2xl font-semibold tracking-tight">
      {{ t('setup.done.title') }}
    </h1>

    <p
      v-if="autoLoginFailed"
      class="text-sm text-muted-foreground"
    >
      {{ t('setup.done.autoLoginFailed') }}
    </p>

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
