<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Separator } from '@/components/ui/separator'
import { SidebarTrigger } from '@/components/ui/sidebar'
import ModeToggle from '@/components/theme/ModeToggle.vue'
import { getRouteTitleKey, translateRouteTitle } from '@/router/route-title'

const route = useRoute()
const { t } = useI18n()

const title = computed(() => {
  const titleKey = getRouteTitleKey(route)
  return titleKey ? translateRouteTitle(titleKey) : t('app.name')
})
</script>

<template>
  <header
    class="flex shrink-0 items-center gap-2 border-be transition-[width,height] ease-linear block-(--header-height) group-has-data-[collapsible=icon]/sidebar-wrapper:block-(--header-height)"
  >
    <div class="flex items-center gap-1 px-4 inline-full lg:gap-2 lg:px-6">
      <SidebarTrigger class="-ms-1" />
      <Separator
        orientation="vertical"
        class="mx-2 data-[orientation=vertical]:block-4"
      />
      <h1 class="text-base font-medium">
        {{ title }}
      </h1>
      <div class="ms-auto flex items-center gap-2">
        <ModeToggle />
      </div>
    </div>
  </header>
</template>
