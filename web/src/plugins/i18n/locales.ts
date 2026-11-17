import { defaultLocale, supportedLocales } from '@/locales'
import type { AppLocale } from '@/locales'

export { fallbackLocale, supportedLocales } from '@/locales'
export type { AppLocale } from '@/locales'

const STORAGE_KEY = 'app.locale'
const supportedLocaleValues = new Set<string>(
  supportedLocales.map((locale) => locale.value),
)

// Accepts `unknown` so callers can narrow untrusted values (storage, UI event payloads)
// without a type assertion.
export function isSupportedLocale(locale: unknown): locale is AppLocale {
  return typeof locale === 'string' && supportedLocaleValues.has(locale)
}

/**
 * Maps a browser language tag (`en`, `zh-Hans-CN`, …) onto a supported locale: exact
 * match first, then the primary subtag (`en-GB` → `en-US`, `zh-Hans-CN` → `zh-CN`).
 * No alias table, so adding a language needs no edit here.
 */
function normalizeLocale(locale: string): AppLocale | null {
  const normalizedLocale = locale.toLowerCase()
  const exactMatch = supportedLocales.find(
    (candidate) => candidate.value.toLowerCase() === normalizedLocale,
  )

  if (exactMatch !== undefined) {
    return exactMatch.value
  }

  const primaryLocale = normalizedLocale.split('-')[0]
  const primaryMatch = supportedLocales.find((candidate) => {
    const value = candidate.value.toLowerCase()
    return value === primaryLocale || value.startsWith(`${primaryLocale}-`)
  })

  return primaryMatch?.value ?? null
}

export function resolveInitialLocale(): AppLocale {
  if (typeof window !== 'undefined') {
    const persistedLocale = window.localStorage.getItem(STORAGE_KEY)
    if (isSupportedLocale(persistedLocale)) {
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
