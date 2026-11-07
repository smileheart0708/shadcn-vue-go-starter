import type { ComputedRef } from 'vue'
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { i18n } from '@/plugins/i18n'
import { getRouteTitleKey, translateRouteTitle } from '@/router/route-title'

// Shared by the shell h1 and the in-page h2, so the two headings cannot drift apart.
export function useRouteTitle(): ComputedRef<string> {
  const route = useRoute()

  return computed(() => {
    const titleKey = getRouteTitleKey(route)
    return titleKey ? translateRouteTitle(titleKey) : i18n.global.t('app.name')
  })
}
