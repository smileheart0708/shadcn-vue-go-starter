import { watchEffect } from 'vue'
import type { Router } from 'vue-router'
import { i18n } from '@/plugins/i18n'
import { getRouteTitleKey, translateRouteTitle } from '@/router/route-title'

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
          page: translateRouteTitle(titleKey),
        })
      : i18n.global.t('app.title', { name: appName })
  }

  // `t()` reads the composer's `locale` ref, so this effect already re-runs on every
  // language switch; the route is tracked through `router.currentRoute`.
  watchEffect(updateDocumentTitle)
}
