import type { RouteRecordRaw } from 'vue-router'
import { createRouter, createWebHistory } from 'vue-router'
import { CAPABILITY } from '@/lib/auth/capabilities.generated'
import { installAuthGuard } from '@/middleware/auth'
import { installRouteLoading } from '@/router/route-loading'
import type { RouteTitleKey } from '@/router/route-title'

const AppShellLayout = async () => import('@/layouts/AppShellLayout.vue')
const BlankLayout = async () => import('@/layouts/BlankLayout.vue')
const Dashboard = async () => import('@/views/DashboardView.vue')
const Setup = async () => import('@/views/SetupView.vue')
const Login = async () => import('@/views/LoginView.vue')
const Register = async () => import('@/views/RegisterView.vue')
const Settings = async () => import('@/views/SettingsView.vue')
const AdminSettings = async () => import('@/views/AdminSettingsView.vue')
const AdminUsers = async () => import('@/views/AdminUsersView.vue')
const SystemLogs = async () => import('@/views/SystemLogsView.vue')
const NotFound = async () => import('@/views/NotFoundView.vue')

function defineAppShellRoute<
  T extends RouteRecordRaw & { meta: { titleKey: RouteTitleKey } },
>(route: T) {
  return route
}

const appShellRoutes = [
  defineAppShellRoute({
    path: 'dashboard',
    name: 'dashboard',
    component: Dashboard,
    meta: { titleKey: 'route.dashboard', requiresAuth: true },
  }),
  defineAppShellRoute({
    path: 'settings',
    name: 'settings',
    component: Settings,
    meta: { titleKey: 'route.settings', requiresAuth: true },
  }),
  defineAppShellRoute({
    path: 'system-configs',
    name: 'system-config',
    component: AdminSettings,
    meta: {
      titleKey: 'route.systemConfig',
      requiresAuth: true,
      requiredCapabilities: [CAPABILITY.systemSettingsRead],
      maskUnauthorizedAsNotFound: true,
    },
  }),
  defineAppShellRoute({
    path: 'users',
    name: 'users',
    component: AdminUsers,
    meta: {
      titleKey: 'route.adminUsers',
      requiresAuth: true,
      requiredCapabilities: [CAPABILITY.managementUsersRead],
      maskUnauthorizedAsNotFound: true,
    },
  }),
  defineAppShellRoute({
    path: 'system-logs',
    name: 'system-logs',
    component: SystemLogs,
    meta: {
      titleKey: 'route.systemLogs',
      requiresAuth: true,
      requiredCapabilities: [CAPABILITY.managementSystemLogsRead],
      maskUnauthorizedAsNotFound: true,
    },
  }),
] satisfies RouteRecordRaw[]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: AppShellLayout,
      children: [
        // The site root lands on the workspace; the auth guard sends guests to
        // /login and uninitialized installs to /setup.
        { path: '', redirect: { name: 'dashboard' } },
        ...appShellRoutes,
      ],
    },
    {
      path: '/setup',
      name: 'setup',
      component: Setup,
      meta: { titleKey: 'route.setup' },
    },
    {
      path: '/login',
      name: 'login',
      component: Login,
      meta: { titleKey: 'route.login', guestOnly: true },
    },
    {
      path: '/register',
      name: 'register',
      component: Register,
      meta: { titleKey: 'route.register', guestOnly: true },
    },
    {
      path: '/:pathMatch(.*)*',
      component: BlankLayout,
      children: [
        {
          path: '',
          name: 'not-found',
          component: NotFound,
          meta: { titleKey: 'route.notFound' },
        },
      ],
    },
  ],
})

installAuthGuard(router)
installRouteLoading(router)

export default router
