import { computed } from 'vue'
import { defineStore } from 'pinia'
import { i18n, setLocale as applyLocale } from '@/plugins/i18n'
import type { AppLocale } from '@/plugins/i18n/locales'

export const useLocaleStore = defineStore('locale', () => {
  // Read through to vue-i18n instead of mirroring the value into a local ref. The i18n
  // instance is the single source of truth; a mirrored copy silently drifts whenever
  // setLocale is called from outside this store.
  const locale = computed<AppLocale>(() => i18n.global.locale.value)

  function setLocale(nextLocale: AppLocale) {
    if (locale.value === nextLocale) {
      return
    }

    applyLocale(nextLocale)
  }

  return { locale, setLocale }
})
