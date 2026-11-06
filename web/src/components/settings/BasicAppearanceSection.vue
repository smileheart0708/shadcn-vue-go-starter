<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { AppLocale } from '@/plugins/i18n/locales'
import type { ThemePreference } from '@/stores/theme'
import type { LocaleOption } from '@/utils/settings/preferences'
import SettingsGroup from '@/components/common/SettingsGroup.vue'
import SettingsRow from '@/components/common/SettingsRow.vue'

const props = defineProps<{
  theme: ThemePreference
  locale: AppLocale
  localeOptions: LocaleOption[]
}>()

const emit = defineEmits<{
  'update:theme': [value: unknown]
  'update:locale': [value: unknown]
}>()

const { t } = useI18n()
</script>

<template>
  <SettingsGroup :title="t('settings.basic.appearance')">
    <SettingsRow :title="t('settings.basic.theme')">
      <Tabs
        :model-value="props.theme"
        @update:model-value="emit('update:theme', $event)"
      >
        <TabsList>
          <TabsTrigger value="light">{{ t('common.theme.light') }}</TabsTrigger>
          <TabsTrigger value="dark">{{ t('common.theme.dark') }}</TabsTrigger>
          <TabsTrigger value="system">{{
            t('common.theme.system')
          }}</TabsTrigger>
        </TabsList>
      </Tabs>
    </SettingsRow>

    <SettingsRow :title="t('settings.basic.language')">
      <Select
        :model-value="props.locale"
        @update:model-value="emit('update:locale', $event)"
      >
        <SelectTrigger class="inline-full sm:inline-48">
          <SelectValue :placeholder="t('settings.basic.selectLanguage')" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem
            v-for="localeOption in props.localeOptions"
            :key="localeOption.value"
            :value="localeOption.value"
          >
            {{ localeOption.label }}
          </SelectItem>
        </SelectContent>
      </Select>
    </SettingsRow>
  </SettingsGroup>
</template>
