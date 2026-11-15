<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { AppShellNavItem } from '@/components/app-shell/navigation'
import { SidebarMenuButton } from '@/components/ui/sidebar'

const props = defineProps<{ item: AppShellNavItem }>()

const route = useRoute()
const router = useRouter()
const isActive = computed(() => {
  const target = router.resolve(props.item.to)

  return (
    route.path === target.path ||
    (target.path !== '/' && route.path.startsWith(`${target.path}/`))
  )
})
</script>

<template>
  <SidebarMenuButton
    as-child
    :is-active="isActive"
  >
    <RouterLink :to="item.to">
      <component
        :is="item.icon"
        v-if="item.icon"
      />
      <span>{{ item.title }}</span>
    </RouterLink>
  </SidebarMenuButton>
</template>
