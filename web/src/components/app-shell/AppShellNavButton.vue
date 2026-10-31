<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { AppShellNavItem } from '@/components/app-shell/navigation'
import { SidebarMenuButton } from '@/components/ui/sidebar'

const props = defineProps<{ item: AppShellNavItem }>()

const route = useRoute()
const router = useRouter()
const navigationTarget = computed(() => props.item.to ?? '')

const isActive = computed(() => {
  if (props.item.to === undefined || props.item.disabled === true) {
    return false
  }

  const target = router.resolve(props.item.to)

  return (
    route.path === target.path ||
    (target.path !== '/' && route.path.startsWith(`${target.path}/`))
  )
})

const hasTarget = computed(() => props.item.to !== undefined)
const isDisabled = computed(
  () => props.item.disabled === true || !hasTarget.value,
)
</script>

<template>
  <SidebarMenuButton
    v-if="hasTarget && !isDisabled"
    as-child
    :is-active="isActive"
  >
    <RouterLink :to="navigationTarget">
      <component
        :is="item.icon"
        v-if="item.icon"
      />
      <span>{{ item.title }}</span>
    </RouterLink>
  </SidebarMenuButton>

  <SidebarMenuButton
    v-else
    :disabled="isDisabled"
    class="text-sidebar-foreground/70 disabled:pointer-events-none disabled:opacity-50"
  >
    <component
      :is="item.icon"
      v-if="item.icon"
    />
    <span>{{ item.title }}</span>
  </SidebarMenuButton>
</template>
