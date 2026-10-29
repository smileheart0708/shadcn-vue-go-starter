import { createI18n } from 'vue-i18n'
import enUS from '@/locales/en-US'
import type { MessageSchema } from '@/locales/schema'
import zhCN from '@/locales/zh-CN'
import {
  applyDocumentLanguage,
  fallbackLocale,
  persistLocale,
  resolveInitialLocale,
  type AppLocale,
} from '@/plugins/i18n/locales'

const messages: Record<AppLocale, MessageSchema> = {
  'zh-CN': zhCN,
  'en-US': enUS,
}

const initialLocale = resolveInitialLocale()

applyDocumentLanguage(initialLocale)

// The third generic (Legacy) must be an explicit `false`, not just the `legacy: false`
// option: vue-i18n keys `I18n['global']` off that type parameter. Left to inference it
// widens to `boolean`, `global` degrades to the Legacy `VueI18n` shape, and `global.locale`
// types as a bare string — which then makes the correct `.value` access below a type error.
// At runtime the option is still what selects the Composition API: vue-i18n 11 defaults to
// the Legacy API and only drops `legacy` in v12, which is also when the suppression goes.
export const i18n = createI18n<[MessageSchema], AppLocale, false>({
  // eslint-disable-next-line @typescript-eslint/no-deprecated -- runtime switch, v11 only
  legacy: false,
  locale: initialLocale,
  fallbackLocale,
  globalInjection: true,
  messages,
})

export function setLocale(locale: AppLocale) {
  // `global.locale` is a WritableComputedRef under the Composition API. Assigning to the
  // property itself (`i18n.global.locale = locale`) silently replaces the ref with a plain
  // string, permanently severing reactivity — every later switch then no-ops.
  i18n.global.locale.value = locale
  applyDocumentLanguage(locale)
  persistLocale(locale)
}

export default i18n
