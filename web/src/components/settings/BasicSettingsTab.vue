<script setup lang="ts">
import BasicAppearanceSection from '@/components/settings/BasicAppearanceSection.vue'
import DataRefreshIntervalSection from '@/components/settings/DataRefreshIntervalSection.vue'
import { useSettingsPreferences } from '@/composables/settings/useSettingsPreferences'

const {
  theme,
  locale,
  localeOptions,
  pollingIntervalSliderValue,
  currentUserPollingIntervalSeconds,
  handleThemeChange,
  handleLocaleChange,
  handlePollingIntervalSliderChange,
  commitPollingInterval,
} = useSettingsPreferences()
</script>

<template>
  <BasicAppearanceSection
    :theme="theme"
    :locale="locale"
    :locale-options="localeOptions"
    @update:theme="handleThemeChange"
    @update:locale="handleLocaleChange"
  />

  <DataRefreshIntervalSection
    :slider-value="pollingIntervalSliderValue"
    :current-seconds="currentUserPollingIntervalSeconds"
    @update:slider-value="handlePollingIntervalSliderChange"
    @value-commit="commitPollingInterval"
  />
</template>
