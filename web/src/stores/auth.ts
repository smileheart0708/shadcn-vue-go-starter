import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { Router } from 'vue-router'
import type {
  InstallState,
  LoginCredentials,
  PublicAuthConfig,
  RegisterInput,
  SessionResponse,
  SetupInput,
  Viewer,
} from '@/lib/api/auth'
import {
  completeSetup as completeSetupRequest,
  getInstallState,
  getPublicAuthConfig,
  getViewer,
  login as loginRequest,
  logout as logoutRequest,
  refreshSession,
  register as registerRequest,
} from '@/lib/api/auth'
import {
  clearAuthClientHandlers,
  registerAuthClientHandlers,
} from '@/lib/api/client'
import { hasCapability } from '@/lib/auth/authorize'
import type { Capability } from '@/lib/auth/capabilities.generated'
import { clearAuthToken, readAuthToken, writeAuthToken } from '@/lib/auth/token'
import { isSameViewer } from '@/utils/auth/viewer'

export const useAuthStore = defineStore('auth', () => {
  const initialized = ref(false)
  const token = ref<string | null>(readAuthToken())
  const viewer = ref<Viewer | null>(null)
  const installState = ref<InstallState | null>(null)
  const publicAuthConfig = ref<PublicAuthConfig | null>(null)

  const isSetupComplete = computed(
    () => installState.value?.setupCompleted === true,
  )
  const isAuthenticated = computed(
    () => token.value !== null && viewer.value !== null,
  )
  const capabilities = computed(
    () => viewer.value?.authorization.capabilities ?? [],
  )

  let initializePromise: Promise<void> | null = null
  let boundRouter: Router | null = null
  let clientHandlersRegistered = false

  function bindRouter(router: Router) {
    boundRouter = router

    if (clientHandlersRegistered) {
      return
    }

    registerAuthClientHandlers({
      refreshAccessToken: async () => {
        try {
          const response = await refreshSession()
          applySession(response)
          return response.accessToken
        } catch (error) {
          // Any refresh failure drops the in-memory session. A 5xx keeps the server-side
          // cookie, so a later page load can restore it; distinguishing status codes here
          // would change the current logout UX.
          resetSession(false)
          await refreshPublicState()
          throw error
        }
      },
      onUnauthorized: async () => {
        resetSession(false)
        await refreshPublicState()
        await redirectAfterSessionLoss()
      },
    })

    clientHandlersRegistered = true
  }

  function setToken(nextToken: string | null) {
    token.value = nextToken
    if (nextToken !== null) {
      writeAuthToken(nextToken)
      return
    }
    clearAuthToken()
  }

  function applyViewer(nextViewer: Viewer | null) {
    // Background polling returns an equal-but-new object on every tick. Replacing the ref
    // for an unchanged payload would invalidate every viewer-derived computed and re-render
    // the shell (and make a future `watch(viewer)` fire on each tick), so the reference is
    // kept stable while the payload is identical — see `isSameViewer`.
    if (isSameViewer(viewer.value, nextViewer)) {
      return
    }

    viewer.value = nextViewer
  }

  function applySession(response: SessionResponse) {
    setToken(response.accessToken)
    applyViewer(response.viewer)
    initialized.value = true
  }

  function resetSession(markInitialized = true) {
    setToken(null)
    applyViewer(null)
    if (markInitialized) {
      initialized.value = true
    }
  }

  async function refreshPublicState() {
    const [nextInstallState, nextPublicAuthConfig] = await Promise.all([
      getInstallState(),
      getPublicAuthConfig(),
    ])
    installState.value = nextInstallState
    publicAuthConfig.value = nextPublicAuthConfig
  }

  async function refreshViewer(
    options: { signal?: AbortSignal; backgroundRequest?: boolean } = {},
  ) {
    if (token.value === null || !isSetupComplete.value) {
      return null
    }

    const nextViewer = await fetchViewer(options)
    applyViewer(nextViewer)
    initialized.value = true
    return nextViewer
  }

  async function fetchViewer(
    options: { signal?: AbortSignal; backgroundRequest?: boolean } = {},
  ) {
    return getViewer(options)
  }

  async function initialize() {
    if (
      initialized.value &&
      (!isSetupComplete.value || viewer.value !== null)
    ) {
      return
    }

    if (initializePromise !== null) {
      return initializePromise
    }

    initializePromise = (async () => {
      try {
        await refreshPublicState()

        if (!isSetupComplete.value) {
          resetSession(false)
          initialized.value = true
          return
        }

        if (token.value !== null) {
          try {
            applyViewer(await fetchViewer())
            initialized.value = true
            return
          } catch {
            resetSession(false)
          }
        }

        try {
          applySession(await refreshSession())
        } catch {
          // Same refresh contract as refreshAccessToken: any failure ends this page as
          // logged out even though a 5xx leaves the cookie intact for the next load.
          resetSession(true)
        }
      } finally {
        initializePromise = null
      }
    })()

    return initializePromise
  }

  async function completeSetup(input: SetupInput) {
    const response = await completeSetupRequest(input)
    applySession(response)
    installState.value = {
      setupCompleted: true,
      ownerUserId: response.viewer.identity.id,
      completedAt: new Date().toISOString(),
    }
    await refreshPublicState()
    return response
  }

  async function login(credentials: LoginCredentials) {
    const response = await loginRequest(credentials)
    applySession(response)
    return response
  }

  async function register(input: RegisterInput) {
    const response = await registerRequest(input)
    applySession(response)
    return response
  }

  async function logout() {
    try {
      await logoutRequest()
    } finally {
      resetSession(true)
    }
  }

  async function redirectAfterSessionLoss() {
    if (!boundRouter) {
      return
    }

    const currentRoute = boundRouter.currentRoute.value
    if (!isSetupComplete.value) {
      await boundRouter.push({ name: 'setup' }).catch(() => undefined)
      return
    }

    const redirect =
      currentRoute.fullPath !== '/login' ? currentRoute.fullPath : undefined
    await boundRouter
      .push({
        name: 'login',
        query: redirect !== undefined ? { redirect } : undefined,
      })
      .catch(() => undefined)
  }

  function can(capability: Capability) {
    return hasCapability(capabilities.value, capability)
  }

  return {
    initialized,
    token,
    viewer,
    installState,
    publicAuthConfig,
    capabilities,
    isSetupComplete,
    isAuthenticated,
    bindRouter,
    fetchViewer,
    applyViewer,
    refreshPublicState,
    refreshViewer,
    initialize,
    completeSetup,
    login,
    register,
    logout,
    resetSession,
    can,
  }
})

if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    clearAuthClientHandlers()
  })
}
