import { z } from 'zod'
import { authApi, normalizeAPIError } from '@/lib/api/client'
import {
  managedUserActionSchema,
  roleKeySchema,
  userStatusSchema,
  type UserStatus,
} from '@/lib/api/contract.generated'
import { successEnvelopeSchema } from '@/lib/api/envelope'

export const managedUserSchema = z.object({
  id: z.number().int().positive(),
  username: z.string(),
  email: z.email().nullable(),
  avatarUrl: z.string().nullable(),
  role: roleKeySchema,
  status: userStatusSchema,
  lastActiveAt: z.string().nullable(),
  createdAt: z.string(),
  updatedAt: z.string(),
  actions: z.array(managedUserActionSchema),
})

export type ManagedUser = z.infer<typeof managedUserSchema>

export interface ListManagedUsersParams {
  q?: string
  status?: UserStatus | null
  page?: number
  pageSize?: number
}

export interface ManagedUserUpsertInput {
  username: string
  email: string | null
}

export interface ManagedUserCreateInput extends ManagedUserUpsertInput {
  password: string
}

const managementUsersPageSchema = successEnvelopeSchema(
  z.object({
    items: z.array(managedUserSchema),
    page: z.number().int().positive(),
    pageSize: z.number().int().positive(),
    total: z.number().int().nonnegative(),
  }),
)

const managedUserEnvelopeSchema = successEnvelopeSchema(managedUserSchema)

export async function listAdminUsers(params: ListManagedUsersParams = {}) {
  try {
    const searchParams = new URLSearchParams()
    const query = params.q?.trim()
    if (query !== undefined && query !== '') {
      searchParams.set('q', query)
    }
    if (params.status !== null && params.status !== undefined) {
      searchParams.set('status', params.status)
    }
    if (typeof params.page === 'number') {
      searchParams.set('page', String(params.page))
    }
    if (typeof params.pageSize === 'number') {
      searchParams.set('pageSize', String(params.pageSize))
    }

    const payload = await authApi
      .get('/api/management/users', { searchParams })
      .json<unknown>()
    return managementUsersPageSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function createAdminUser(input: ManagedUserCreateInput) {
  try {
    const payload = await authApi
      .post('/api/management/users', { json: input })
      .json<unknown>()
    return managedUserEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function updateAdminUser(
  id: number,
  input: ManagedUserUpsertInput,
) {
  try {
    const payload = await authApi
      .patch(`/api/management/users/${String(id)}`, { json: input })
      .json<unknown>()
    return managedUserEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function disableAdminUser(id: number) {
  try {
    const payload = await authApi
      .post(`/api/management/users/${String(id)}/disable`)
      .json<unknown>()
    return managedUserEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function enableAdminUser(id: number) {
  try {
    const payload = await authApi
      .post(`/api/management/users/${String(id)}/enable`)
      .json<unknown>()
    return managedUserEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}
