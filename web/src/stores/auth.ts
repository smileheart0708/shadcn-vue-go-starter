import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { Router } from 'vue-router'
import type {
  Capability,
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
import { clearAuthToken, readAuthToken, writeAuthToken } from '@/lib/auth/token'

export const useAuthStore = defineStore('auth', () => {
  const initialized = ref(false)
  const initializing = ref(false)
  const refreshing = ref(false)
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
  const currentRole = computed(() => viewer.value?.authorization.role ?? null)

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
        refreshing.value = true

        try {
          const response = await refreshSession()
          applySession(response)
          return response.accessToken
        } catch (error) {
          resetSession(false)
          await refreshPublicState()
          throw error
        } finally {
          refreshing.value = false
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
      initializing.value = true

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
          resetSession(true)
        }
      } finally {
        initializing.value = false
        initializePromise = null
      }
    })()

    return initializePromise
  }

  async function completeSetup(input: SetupInput) {
    const response = await completeSetupRequest(input)
    applySession(response)
    installState.value = {
      setupState: 'completed',
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
    return capabilities.value.includes(capability)
  }

  return {
    initialized,
    initializing,
    refreshing,
    token,
    viewer,
    installState,
    publicAuthConfig,
    capabilities,
    currentRole,
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
