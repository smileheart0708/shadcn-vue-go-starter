import {
  localeNames,
  supportedLocales,
  type AppLocale,
} from '@/plugins/i18n/locales'
import type { ThemePreference } from '@/stores/theme'

export interface LocaleOption {
  value: AppLocale
  label: string
}

export function isThemePreference(value: unknown): value is ThemePreference {
  return value === 'light' || value === 'dark' || value === 'system'
}

export function getLocaleOptions(): LocaleOption[] {
  return supportedLocales.map((value) => ({
    value,
    label: localeNames[value],
  }))
}

export function normalizeOptionalEmail(value: string): string | null {
  const email = value.trim()
  return email.length > 0 ? email : null
}
