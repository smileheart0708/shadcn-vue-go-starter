import ky, { HTTPError } from 'ky'
import type {
  AfterResponseHook,
  BeforeRequestHook,
  Hooks,
  RetryOptions,
} from 'ky'
import { readAuthToken } from '@/lib/auth/token'
import { BACKGROUND_REQUEST_HEADER, trackedFetch } from '@/router/route-loading'

interface APIErrorPayload {
  error?: { code?: string; message?: string }
}

interface APIClientContext {
  skipAuthRefresh?: boolean
  backgroundRequest?: boolean
}

type RefreshAccessTokenHandler = () => Promise<string>
type UnauthorizedHandler = () => Promise<void> | void

let refreshAccessTokenHandler: RefreshAccessTokenHandler | null = null
let unauthorizedHandler: UnauthorizedHandler | null = null
let refreshPromise: Promise<string> | null = null

export class APIError extends Error {
  status: number
  code?: string
  payload?: unknown

  constructor(
    status: number,
    message: string,
    code?: string,
    payload?: unknown,
  ) {
    super(message)
    this.name = 'APIError'
    this.status = status
    this.code = code
    this.payload = payload
  }
}

export function registerAuthClientHandlers(handlers: {
  refreshAccessToken: RefreshAccessTokenHandler
  onUnauthorized: UnauthorizedHandler
}) {
  refreshAccessTokenHandler = handlers.refreshAccessToken
  unauthorizedHandler = handlers.onUnauthorized
}

export function clearAuthClientHandlers() {
  refreshAccessTokenHandler = null
  unauthorizedHandler = null
  refreshPromise = null
}

const applySharedRequestHeaders: BeforeRequestHook = ({ request, options }) => {
  const context = readAPIClientContext(options.context)

  if (!request.headers.has('Accept')) {
    request.headers.set('Accept', 'application/json')
  }

  if (context.backgroundRequest === true) {
    request.headers.set(BACKGROUND_REQUEST_HEADER, '1')
  }
}

const applyAuthHeader: BeforeRequestHook = ({ request }) => {
  const token = readAuthToken()
  if (token !== null && !request.headers.has('Authorization')) {
    request.headers.set('Authorization', `Bearer ${token}`)
  }
}

const refreshUnauthorizedResponse: AfterResponseHook = async ({
  request,
  options,
  response,
  retryCount,
}) => {
  const context = readAPIClientContext(options.context)
  if (
    response.status !== 401 ||
    retryCount > 0 ||
    context.skipAuthRefresh === true ||
    isAuthLifecycleRequest(request)
  ) {
    return response
  }

  if (!refreshAccessTokenHandler) {
    return response
  }

  let refreshedToken: string
  try {
    refreshedToken = await refreshAccessTokenSingleFlight()
  } catch {
    await unauthorizedHandler?.()
    return response
  }

  return ky.retry({
    code: 'AUTH_TOKEN_REFRESHED',
    delay: 0,
    request: createRetriedAuthRequest(request, refreshedToken),
  })
}

const sharedBeforeRequestHooks: BeforeRequestHook[] = [
  applySharedRequestHeaders,
]

const sharedHooks: Hooks = {
  beforeRequest: sharedBeforeRequestHooks,
}

const authRefreshRetry: RetryOptions = {
  limit: 1,
  methods: [],
  statusCodes: [],
  delay: () => 0,
}

export const baseApi = ky.create({
  retry: 0,
  credentials: 'same-origin',
  // Routes every API call through the loading-bar counter without patching global fetch.
  fetch: trackedFetch,
  hooks: sharedHooks,
})

export const authApi = baseApi.extend({
  retry: authRefreshRetry,
  hooks: {
    beforeRequest: [applyAuthHeader],
    afterResponse: [refreshUnauthorizedResponse],
  },
})

async function toAPIError(error: HTTPError): Promise<APIError> {
  const payload = await readResponsePayload(error.response.clone())
  const errorPayload = isAPIErrorPayload(payload) ? payload : undefined

  return new APIError(
    error.response.status,
    errorPayload?.error?.message ??
      `Request failed with status ${String(error.response.status)}`,
    errorPayload?.error?.code,
    payload,
  )
}

export async function normalizeAPIError(error: unknown): Promise<never> {
  if (error instanceof APIError) {
    throw error
  }

  if (error instanceof HTTPError) {
    throw await toAPIError(error)
  }

  throw error
}

async function refreshAccessTokenSingleFlight(): Promise<string> {
  if (refreshPromise !== null) {
    return refreshPromise
  }

  if (!refreshAccessTokenHandler) {
    throw new Error('No refresh handler registered.')
  }

  refreshPromise = refreshAccessTokenHandler().finally(() => {
    refreshPromise = null
  })

  return refreshPromise
}

async function readResponsePayload(response: Response): Promise<unknown> {
  const contentType = response.headers.get('Content-Type') ?? ''
  if (contentType.includes('application/json')) {
    return response.json()
  }

  const text = await response.text()
  if (text.length === 0) {
    return undefined
  }

  return text
}

function readAPIClientContext(
  context: Record<string, unknown> | undefined,
): APIClientContext {
  if (context === undefined) {
    return {}
  }

  const { skipAuthRefresh, backgroundRequest } = context
  return {
    skipAuthRefresh: skipAuthRefresh === true,
    backgroundRequest: backgroundRequest === true,
  }
}

function createRetriedAuthRequest(
  request: Request,
  accessToken: string,
): Request {
  const headers = new Headers(request.headers)
  headers.set('Authorization', `Bearer ${accessToken}`)

  return new Request(request, { headers })
}

function isAuthLifecycleRequest(request: Request): boolean {
  const pathname = new URL(request.url, window.location.origin).pathname
  return (
    pathname === '/api/auth/login' ||
    pathname === '/api/auth/refresh' ||
    pathname === '/api/auth/logout'
  )
}

function isAPIErrorPayload(payload: unknown): payload is APIErrorPayload {
  if (!isRecord(payload)) {
    return false
  }

  const { error } = payload
  if (error === undefined) {
    return true
  }

  if (!isRecord(error)) {
    return false
  }

  const { code, message } = error
  return (
    (code === undefined || typeof code === 'string') &&
    (message === undefined || typeof message === 'string')
  )
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
