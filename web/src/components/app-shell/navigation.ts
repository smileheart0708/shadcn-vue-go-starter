import { computed, type Component, type ComputedRef } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Cog, LayoutDashboard, Logs, Settings, Users } from '@lucide/vue'
import { CAPABILITY } from '@/lib/auth/capabilities.generated'
import { useAuthStore } from '@/stores/auth'

export interface AppShellNavItem {
  title: string
  icon?: Component
  to: RouteLocationRaw
}

export interface AppShellNavigation {
  main: AppShellNavItem[]
  management: AppShellNavItem[]
  secondary: AppShellNavItem[]
}

export function useAppShellNavigation(): ComputedRef<AppShellNavigation> {
  const { t } = useI18n()
  const authStore = useAuthStore()

  const canReadSystemSettings = computed(() =>
    authStore.can(CAPABILITY.systemSettingsRead),
  )
  const canReadUsers = computed(() =>
    authStore.can(CAPABILITY.managementUsersRead),
  )
  const canReadSystemLogs = computed(() =>
    authStore.can(CAPABILITY.managementSystemLogsRead),
  )

  return computed(() => ({
    main: [
      {
        title: t('nav.main.dashboard'),
        icon: LayoutDashboard,
        to: { name: 'dashboard' },
      },
    ],
    management: [
      ...(canReadSystemSettings.value
        ? [
            {
              title: t('nav.management.systemConfig'),
              icon: Cog,
              to: { name: 'system-config' },
            },
          ]
        : []),
      ...(canReadUsers.value
        ? [
            {
              title: t('nav.management.users'),
              icon: Users,
              to: { name: 'users' },
            },
          ]
        : []),
      ...(canReadSystemLogs.value
        ? [
            {
              title: t('nav.management.systemLogs'),
              icon: Logs,
              to: { name: 'system-logs' },
            },
          ]
        : []),
    ],
    secondary: [
      {
        title: t('nav.secondary.settings'),
        icon: Settings,
        to: { name: 'settings' },
      },
    ],
  }))
}
