import { z } from 'zod'
import { authApi, baseApi, normalizeAPIError } from '@/lib/api/client'
import { successEnvelopeSchema } from '@/lib/api/envelope'
import { capabilitySchema } from '@/lib/auth/capabilities.generated'

export const roleKeySchema = z.enum(['owner', 'user'])

export const viewerIdentitySchema = z.object({
  id: z.number().int().positive(),
  username: z.string(),
  email: z.email().nullable(),
  avatarUrl: z.string().nullable(),
  status: z.enum(['active', 'disabled']),
})

export const viewerAuthorizationSchema = z.object({
  role: roleKeySchema,
  capabilities: z.array(capabilitySchema),
})

export const viewerSchema = z.object({
  identity: viewerIdentitySchema,
  authorization: viewerAuthorizationSchema,
})

export const installStateSchema = z.object({
  setupState: z.enum(['pending', 'completed']),
  setupCompleted: z.boolean(),
  ownerUserId: z.number().int().positive().nullable(),
  completedAt: z.string().nullable(),
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

export type RoleKey = z.infer<typeof roleKeySchema>
export type ViewerIdentity = z.infer<typeof viewerIdentitySchema>
export type ViewerAuthorization = z.infer<typeof viewerAuthorizationSchema>
export type Viewer = z.infer<typeof viewerSchema>
export type InstallState = z.infer<typeof installStateSchema>
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

export interface SetupInput {
  username: string
  password: string
}

export interface GetViewerOptions {
  signal?: AbortSignal
  backgroundRequest?: boolean
}

const installStateEnvelopeSchema = successEnvelopeSchema(installStateSchema)
const publicAuthConfigEnvelopeSchema = successEnvelopeSchema(
  publicAuthConfigSchema,
)
const sessionEnvelopeSchema = successEnvelopeSchema(sessionPayloadSchema)
const viewerEnvelopeSchema = successEnvelopeSchema(viewerSchema)
const logoutEnvelopeSchema = successEnvelopeSchema(logoutPayloadSchema)

export async function getInstallState() {
  try {
    const payload = await baseApi.get('/api/install/state').json<unknown>()
    return installStateEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function completeSetup(
  input: SetupInput,
): Promise<SessionResponse> {
  try {
    const payload = await baseApi
      .post('/api/install/setup', { json: input })
      .json<unknown>()
    return sessionEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

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
