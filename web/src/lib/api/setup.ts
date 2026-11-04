import { z } from 'zod'
import { baseApi, normalizeAPIError } from '@/lib/api/client'
import { successEnvelopeSchema } from '@/lib/api/envelope'

export const databaseDriverSchema = z.enum(['sqlite', 'postgresql'])
export const installPhaseSchema = z.enum([
  'database_required',
  'finalization_required',
  'completed',
])
export const sslModeSchema = z.enum(['disable', 'require', 'verify-full'])

export const installStateSchema = z.object({
  phase: installPhaseSchema,
  databaseDriver: databaseDriverSchema.nullable(),
  ownerUserId: z.number().int().positive().nullable(),
  completedAt: z.string().nullable(),
})

export type DatabaseDriver = z.infer<typeof databaseDriverSchema>
export type InstallState = z.infer<typeof installStateSchema>
export type SSLMode = z.infer<typeof sslModeSchema>

export interface PostgreSQLConnectionInput {
  host: string
  port: number
  database: string
  username: string
  password: string
  sslMode: SSLMode
}

export type SetupDatabaseInput =
  | { driver: 'sqlite' }
  | { driver: 'postgresql'; postgresql: PostgreSQLConnectionInput }

export interface SetupInput {
  username: string
  password: string
  database?: SetupDatabaseInput
}

const installStateEnvelopeSchema = successEnvelopeSchema(installStateSchema)

export async function getInstallState() {
  try {
    const payload = await baseApi.get('/api/install/state').json<unknown>()
    return installStateEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function completeSetup(input: SetupInput) {
  try {
    const payload = await baseApi
      .post('/api/install/setup', { json: input })
      .json<unknown>()
    return installStateEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}
