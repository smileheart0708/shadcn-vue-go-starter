import { beforeEach, describe, expect, it } from 'vitest'
import { defaultLocale, supportedLocales } from '@/locales'
import { resolveInitialLocale } from './locales'

function setBrowserLanguages(languages: readonly string[]) {
  Object.defineProperty(window.navigator, 'languages', {
    configurable: true,
    value: [...languages],
  })
}

// Fixtures derive from the registry so adding or removing a language pack never needs an
// edit here; `xx-YY` is syntactically valid and can never be a supported pack by accident.
const [defaultLanguage] = defaultLocale.split('-')
const otherSupportedLocale = supportedLocales[supportedLocales.length - 1].value
const UNSUPPORTED_TAG = 'xx-YY'

describe('resolveInitialLocale', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  it('prefers a persisted supported locale over the browser', () => {
    window.localStorage.setItem('app.locale', otherSupportedLocale)
    setBrowserLanguages([`${defaultLanguage}-ZZ`])

    expect(resolveInitialLocale()).toBe(otherSupportedLocale)
  })

  it('ignores a persisted locale that is no longer supported', () => {
    window.localStorage.setItem('app.locale', UNSUPPORTED_TAG)
    setBrowserLanguages([`${defaultLanguage}-ZZ`])

    expect(resolveInitialLocale()).toBe(defaultLocale)
  })

  it('matches a browser tag by its primary subtag', () => {
    setBrowserLanguages([`${defaultLanguage}-ZZ`])

    expect(resolveInitialLocale()).toBe(defaultLocale)
  })

  it('drops script subtags before matching', () => {
    setBrowserLanguages([`${defaultLanguage}-Latn-ZZ`])

    expect(resolveInitialLocale()).toBe(defaultLocale)
  })

  it('falls back to the default locale for unsupported languages', () => {
    setBrowserLanguages([UNSUPPORTED_TAG])

    expect(resolveInitialLocale()).toBe(defaultLocale)
  })
})
