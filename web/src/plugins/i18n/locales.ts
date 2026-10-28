import { z } from 'zod'
import { parseExternalValue } from '@/lib/external-input'

export const supportedLocales = ['zh-CN', 'en-US'] as const
const appLocaleSchema = z.enum(supportedLocales)

export type AppLocale = z.infer<typeof appLocaleSchema>

/** Endonyms for the language switcher — always shown in their own language. */
export const localeNames: Record<AppLocale, string> = {
  'zh-CN': '简体中文',
  'en-US': 'English',
}

export const fallbackLocale: AppLocale = 'en-US'

const STORAGE_KEY = 'app.locale'
const defaultLocale: AppLocale = 'zh-CN'
const localeAliases: Readonly<Partial<Record<string, AppLocale>>> = {
  en: 'en-US',
  'en-us': 'en-US',
  zh: 'zh-CN',
  'zh-cn': 'zh-CN',
}

// Accepts `unknown` so callers can narrow untrusted values (storage, UI event payloads)
// without a type assertion.
export function isSupportedLocale(locale: unknown): locale is AppLocale {
  return parseExternalValue(appLocaleSchema, locale) !== null
}

/** Maps a browser language tag (`en`, `zh-Hans-CN`, …) onto a supported locale. */
function normalizeLocale(locale: string): AppLocale | null {
  const normalizedLocale = locale.toLowerCase()
  const exactMatch = supportedLocales.find(
    (candidate) => candidate.toLowerCase() === normalizedLocale,
  )
  const primaryLocale = normalizedLocale.split('-')[0]

  if (exactMatch) {
    return exactMatch
  }

  return localeAliases[normalizedLocale] ?? localeAliases[primaryLocale] ?? null
}

export function resolveInitialLocale(): AppLocale {
  if (typeof window !== 'undefined') {
    const persistedLocale = parseExternalValue(
      appLocaleSchema,
      window.localStorage.getItem(STORAGE_KEY),
    )
    if (persistedLocale !== null) {
      return persistedLocale
    }
  }

  if (typeof navigator !== 'undefined') {
    const browserLocales =
      navigator.languages.length > 0
        ? navigator.languages
        : [navigator.language]

    for (const candidate of browserLocales) {
      const normalizedLocale = normalizeLocale(candidate)

      if (normalizedLocale !== null) {
        return normalizedLocale
      }
    }
  }

  return defaultLocale
}

export function applyDocumentLanguage(locale: AppLocale) {
  if (typeof document === 'undefined') {
    return
  }

  // The locale keys are already valid BCP 47 tags, so they double as `<html lang>`.
  document.documentElement.lang = locale
}

export function persistLocale(locale: AppLocale) {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.setItem(STORAGE_KEY, locale)
}
