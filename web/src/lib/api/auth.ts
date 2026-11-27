import { z } from 'zod'
import { authApi, baseApi, normalizeAPIError } from '@/lib/api/client'
import { roleKeySchema, userStatusSchema } from '@/lib/api/contract.generated'
import { successEnvelopeSchema } from '@/lib/api/envelope'
import { capabilitySchema } from '@/lib/auth/capabilities.generated'
import { runWithRefreshLock } from '@/lib/auth/refresh-lock'

export const viewerIdentitySchema = z.object({
  id: z.number().int().positive(),
  username: z.string(),
  // The server enforces the format on write. Reading the stored value as an opaque
  // string keeps one legacy row from making the whole viewer response unparsable.
  email: z.string().nullable(),
  avatarUrl: z.string().nullable(),
  status: userStatusSchema,
})

export const viewerAuthorizationSchema = z.object({
  role: roleKeySchema,
  capabilities: z.array(capabilitySchema),
})

export const viewerSchema = z.object({
  identity: viewerIdentitySchema,
  authorization: viewerAuthorizationSchema,
})

export const publicAuthConfigSchema = z.object({
  registrationEnabled: z.boolean(),
})

const sessionPayloadSchema = z.object({
  accessToken: z.string(),
  tokenType: z.literal('Bearer'),
  expiresAt: z.string(),
  viewer: viewerSchema,
})

const logoutPayloadSchema = z.object({
  loggedOut: z.boolean(),
})

export type ViewerIdentity = z.infer<typeof viewerIdentitySchema>
export type ViewerAuthorization = z.infer<typeof viewerAuthorizationSchema>
export type Viewer = z.infer<typeof viewerSchema>
export type PublicAuthConfig = z.infer<typeof publicAuthConfigSchema>
export type SessionResponse = z.infer<typeof sessionPayloadSchema>

export interface LoginCredentials {
  identifier: string
  password: string
}

export interface RegisterInput {
  username: string
  email: string | null
  password: string
}

export interface GetViewerOptions {
  signal?: AbortSignal
  backgroundRequest?: boolean
}

const publicAuthConfigEnvelopeSchema = successEnvelopeSchema(
  publicAuthConfigSchema,
)
const sessionEnvelopeSchema = successEnvelopeSchema(sessionPayloadSchema)
const viewerEnvelopeSchema = successEnvelopeSchema(viewerSchema)
const logoutEnvelopeSchema = successEnvelopeSchema(logoutPayloadSchema)

export async function getPublicAuthConfig() {
  try {
    const payload = await baseApi.get('/api/auth/public-config').json<unknown>()
    return publicAuthConfigEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function login(
  credentials: LoginCredentials,
): Promise<SessionResponse> {
  try {
    const payload = await baseApi
      .post('/api/auth/login', { json: credentials })
      .json<unknown>()
    return sessionEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function register(input: RegisterInput): Promise<SessionResponse> {
  try {
    const payload = await baseApi
      .post('/api/auth/register', { json: input })
      .json<unknown>()
    return sessionEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function refreshSession(): Promise<SessionResponse> {
  // Every refresh path goes through here, including the startup call in the auth
  // store, so the tab lock also covers boot-time refreshes that never touch authApi.
  return runWithRefreshLock(refreshSessionOnce)
}

// Stays on baseApi: authApi's 401 hook calls refreshSession again, and Web Locks are not
// reentrant per origin, so moving this call would deadlock the tab against its own lock.
async function refreshSessionOnce(): Promise<SessionResponse> {
  try {
    const payload = await baseApi
      .post('/api/auth/refresh', {
        context: {
          skipAuthRefresh: true,
        },
      })
      .json<unknown>()
    return sessionEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function logout() {
  try {
    const payload = await baseApi
      .post('/api/auth/logout', {
        context: {
          skipAuthRefresh: true,
        },
      })
      .json<unknown>()
    return logoutEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function getViewer(options: GetViewerOptions = {}) {
  try {
    const payload = await authApi
      .get('/api/auth/me', {
        signal: options.signal,
        context: {
          backgroundRequest: options.backgroundRequest === true,
        },
      })
      .json<unknown>()
    return viewerEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}
