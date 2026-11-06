<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Slider } from '@/components/ui/slider'
import {
  POLLING_INTERVAL_MAX_SECONDS,
  POLLING_INTERVAL_MIN_SECONDS,
} from '@/stores/polling'
import SettingsGroup from '@/components/common/SettingsGroup.vue'
import SettingsRow from '@/components/common/SettingsRow.vue'

const props = defineProps<{
  sliderValue: number[]
  currentSeconds: number
}>()

const emit = defineEmits<{
  'update:slider-value': [value: number[] | undefined]
  'value-commit': [value: number[] | undefined]
}>()

const { t } = useI18n()
</script>

<template>
  <SettingsGroup :title="t('settings.basic.dataRefresh')">
    <SettingsRow
      :title="t('settings.basic.dataRefreshInterval')"
      :description="
        t('settings.basic.dataRefreshIntervalDesc', {
          seconds: props.currentSeconds,
        })
      "
    >
      <Slider
        class="inline-full sm:inline-64"
        :thumb-label="t('settings.basic.dataRefreshInterval')"
        :model-value="props.sliderValue"
        :min="POLLING_INTERVAL_MIN_SECONDS"
        :max="POLLING_INTERVAL_MAX_SECONDS"
        :step="1"
        @update:model-value="emit('update:slider-value', $event)"
        @value-commit="emit('value-commit', $event)"
      />
    </SettingsRow>
  </SettingsGroup>
</template>
