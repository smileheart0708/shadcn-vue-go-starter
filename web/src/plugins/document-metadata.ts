import { watchEffect } from 'vue'
import type { Router } from 'vue-router'
import { i18n } from '@/plugins/i18n'
import { getRouteTitleKey } from '@/router/route-title'

export function installDocumentMetadata(router: Router) {
  if (typeof document === 'undefined') {
    return
  }

  const updateDocumentTitle = () => {
    const titleKey = getRouteTitleKey(router.currentRoute.value)
    const appName = i18n.global.t('app.name')

    document.title = titleKey
      ? i18n.global.t('app.titleWithPage', {
          name: appName,
          page: i18n.global.t(titleKey),
        })
      : i18n.global.t('app.title', { name: appName })
  }

  watchEffect(() => {
    // Touch the locale ref's value (not the ref object) so the effect re-runs on every
    // language switch; `t()` alone would not register a dependency on the active locale.
    void i18n.global.locale.value
    updateDocumentTitle()
  })
}
