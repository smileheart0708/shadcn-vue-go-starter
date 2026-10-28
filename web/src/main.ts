import { createApp } from 'vue'
import './style.css'
import 'vue-sonner/style.css'
import App from './App.vue'
import { installDocumentMetadata } from './plugins/document-metadata'
import i18n from './plugins/i18n'
import router from './router'
import { useAuthStore } from './stores/auth'
import pinia from './stores/pinia'

async function enableMocking() {
  if (!import.meta.env.DEV || !__VITE_API_MOCKING__) {
    return
  }

  const [{ network }, { handlers }, { initializeMockDevSession }] =
    await Promise.all([
      import('virtual:msw'),
      import('@/mocks/handlers'),
      import('@/mocks/dev-session'),
    ])

  network.configure({ handlers, onUnhandledFrame: 'bypass' })
  await network.enable()
  initializeMockDevSession()
}

async function bootstrap() {
  await enableMocking()

  const app = createApp(App)
  const authStore = useAuthStore(pinia)

  app.use(pinia)
  // Restore auth state before the router starts its first navigation.
  authStore.bindRouter(router)
  await authStore.initialize()

  app.use(router)
  app.use(i18n)

  installDocumentMetadata(router)

  await router.isReady()
  app.mount('#app')
}

void bootstrap()
