import { supportedLocales, type AppLocale } from '@/plugins/i18n/locales'
import type { ThemePreference } from '@/stores/theme'

export interface LocaleOption {
  value: AppLocale
  label: string
}

export function isThemePreference(value: unknown): value is ThemePreference {
  return value === 'light' || value === 'dark' || value === 'system'
}

export function getLocaleOptions(): readonly LocaleOption[] {
  return supportedLocales
}
