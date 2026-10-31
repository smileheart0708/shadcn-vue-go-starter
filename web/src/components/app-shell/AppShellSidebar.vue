<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppShellNavGroup from '@/components/app-shell/AppShellNavGroup.vue'
import AppShellNavUser from '@/components/app-shell/AppShellNavUser.vue'
import { useAppShellNavigation } from '@/components/app-shell/navigation'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
} from '@/components/ui/sidebar'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()
const navigation = useAppShellNavigation()

// No fallback role: before the viewer is loaded the sidebar must not claim to be
// the owner, so the label resolves through the "unknown" branch instead.
const currentUser = computed(() => ({
  username: authStore.viewer?.identity.username ?? '',
  email: authStore.viewer?.identity.email ?? null,
  avatarUrl: authStore.viewer?.identity.avatarUrl ?? null,
  role: authStore.viewer?.authorization.role ?? null,
}))

async function handleLogout() {
  await authStore.logout()
  await router.push({ name: 'login' })
}
</script>

<template>
  <Sidebar collapsible="offcanvas">
    <SidebarHeader>
      <RouterLink
        to="/"
        class="flex items-center gap-2 px-2 transition-opacity hover:opacity-80"
      >
        <img
          src="/logo.svg"
          class="block-8 inline-8"
          alt="Logo"
        />
        <span class="text-base font-semibold">web</span>
      </RouterLink>
    </SidebarHeader>
    <SidebarContent>
      <AppShellNavGroup :items="navigation.main" />
      <AppShellNavGroup
        :items="navigation.management"
        :label="t('nav.management.label')"
      />
      <AppShellNavGroup
        :items="navigation.secondary"
        class="mbs-auto"
      />
    </SidebarContent>
    <SidebarFooter>
      <AppShellNavUser
        :user="currentUser"
        @logout="handleLogout"
      />
    </SidebarFooter>
  </Sidebar>
</template>
