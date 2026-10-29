<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Label } from '@/components/ui/label'
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
import SettingsSectionCard from '@/components/settings/common/SettingsSectionCard.vue'

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
  <SettingsSectionCard
    :title="t('settings.basic.theme')"
    :description="t('settings.basic.themeDesc')"
    content-class="space-y-6"
  >
    <div class="space-y-2">
      <Label>{{ t('settings.basic.colorTheme') }}</Label>
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
    </div>

    <div class="space-y-2">
      <Label>{{ t('settings.basic.language') }}</Label>
      <Select
        :model-value="props.locale"
        @update:model-value="emit('update:locale', $event)"
      >
        <SelectTrigger>
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
    </div>
  </SettingsSectionCard>
</template>
