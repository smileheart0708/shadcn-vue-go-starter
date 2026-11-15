import { createApp } from 'vue'
import './style.css'
import 'vue-sonner/style.css'
import App from './App.vue'
import StartupFailure from './components/app/StartupFailure.vue'
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

  // The store is created before the app so a startup failure never leaves a half-mounted
  // shell behind. `bindRouter` is called explicitly because the store is not created through
  // the app's injection context yet.
  const authStore = useAuthStore(pinia)
  authStore.bindRouter(router)

  // Every route guard and page reads `installState` / `publicAuthConfig` loaded here. This call
  // no longer throws — a failure is recorded on the store — but the failure still has to be
  // surfaced: mounting anyway would route a broken backend to the setup wizard and tell the
  // user their installation is empty.
  await authStore.initialize()
  if (authStore.publicStateFailed) {
    mountStartupFailure()
    return
  }

  const app = createApp(App)

  app.use(pinia)
  app.use(router)
  app.use(i18n)

  installDocumentMetadata(router)

  await router.isReady()
  app.mount('#app')
}

// Rendered instead of the shell when the first request fails. Deliberately a separate app:
// the router guard would redirect based on state that never loaded, and mounting the real app
// would replace this screen with an empty layout.
function mountStartupFailure() {
  const failureApp = createApp(StartupFailure)
  failureApp.use(i18n)
  failureApp.mount('#app')
}

// Last resort for anything thrown before the app mounts (mock setup, an unresolvable initial
// route). Without it the rejection only reaches the console and the user gets a blank document.
void bootstrap().catch((error: unknown) => {
  console.error('application bootstrap failed', error)
  mountStartupFailure()
})
