import { HttpResponse } from 'msw'
import type { RoleKey, ViewerIdentity } from '@/lib/api/auth'

/**
 * In-memory backend state for `pnpm dev` with `VITE_API_MOCKING=true`.
 *
 * Domain types are imported from `lib/api/auth` rather than redeclared so that a schema
 * change breaks the mocks at compile time instead of silently drifting from the client
 * contract they emulate.
 */

type UserStatus = ViewerIdentity['status']

export interface MockUser {
  id: number
  username: string
  email: string | null
  avatarUrl: string | null
  role: RoleKey
  status: UserStatus
  password: string
  lastActiveAt: string | null
  createdAt: string
  updatedAt: string
}

interface MockSession {
  userId: number
  accessToken: string
  expiresAt: string
}

interface MockState {
  installState: {
    setupCompleted: boolean
    ownerUserId: number | null
    completedAt: string | null
  }
  accountPolicies: {
    publicRegistrationEnabled: boolean
    selfServiceAccountDeletionEnabled: boolean
  }
  users: MockUser[]
  session: MockSession | null
  nextUserId: number
  nextLogId: number
}

const MOCK_OWNER_USER_ID = 1
const MOCK_ACTIVE_USER_ID = 2
const MOCK_DISABLED_USER_ID = 3
const MOCK_FIXTURE_CREATED_AT = '2026-01-01T00:00:00.000Z'
const MOCK_FIXTURE_UPDATED_AT = '2026-01-02T00:00:00.000Z'

export const MOCK_DEV_ACCESS_TOKEN = 'mock-owner-access-token'

export function nowISO() {
  return new Date().toISOString()
}

export function createMockSession(
  userId: number,
  accessToken: string,
): MockSession {
  return {
    userId,
    accessToken,
    expiresAt: new Date(Date.now() + 10 * 60_000).toISOString(),
  }
}

function createInitialState(): MockState {
  const users: MockUser[] = [
    {
      id: MOCK_OWNER_USER_ID,
      username: 'owner',
      email: 'owner@example.com',
      avatarUrl: null,
      role: 'owner',
      status: 'active',
      password: 'owner1234',
      lastActiveAt: MOCK_FIXTURE_UPDATED_AT,
      createdAt: MOCK_FIXTURE_CREATED_AT,
      updatedAt: MOCK_FIXTURE_UPDATED_AT,
    },
    {
      id: MOCK_ACTIVE_USER_ID,
      username: 'member',
      email: 'member@example.com',
      avatarUrl: null,
      role: 'user',
      status: 'active',
      password: 'member1234',
      lastActiveAt: null,
      createdAt: MOCK_FIXTURE_CREATED_AT,
      updatedAt: MOCK_FIXTURE_UPDATED_AT,
    },
    {
      id: MOCK_DISABLED_USER_ID,
      username: 'disabled',
      email: 'disabled@example.com',
      avatarUrl: null,
      role: 'user',
      status: 'disabled',
      password: 'disabled1234',
      lastActiveAt: null,
      createdAt: MOCK_FIXTURE_CREATED_AT,
      updatedAt: MOCK_FIXTURE_UPDATED_AT,
    },
  ]

  return {
    installState: {
      setupCompleted: true,
      ownerUserId: MOCK_OWNER_USER_ID,
      completedAt: MOCK_FIXTURE_CREATED_AT,
    },
    accountPolicies: {
      publicRegistrationEnabled: false,
      selfServiceAccountDeletionEnabled: false,
    },
    users,
    session: createMockSession(MOCK_OWNER_USER_ID, MOCK_DEV_ACCESS_TOKEN),
    nextUserId: 4,
    nextLogId: 4,
  }
}

export const state = createInitialState()

export function jsonSuccess(data: unknown, init?: ResponseInit) {
  return HttpResponse.json({ success: true, data }, init)
}

export function jsonError(status: number, code: string, message: string) {
  return HttpResponse.json(
    { success: false, error: { code, message } },
    { status },
  )
}

export function getPublicAuthConfig() {
  return {
    registrationEnabled: state.accountPolicies.publicRegistrationEnabled,
  }
}

export function usernameTaken(username: string, excludeUserID?: number) {
  return state.users.some(
    (user) =>
      user.username.toLowerCase() === username.toLowerCase() &&
      user.id !== excludeUserID,
  )
}

export function emailTaken(email: string | null, excludeUserID?: number) {
  if (email === null || email === '') {
    return false
  }

  return state.users.some(
    (user) =>
      user.email?.toLowerCase() === email.toLowerCase() &&
      user.id !== excludeUserID,
  )
}

/**
 * Returns `unknown` (not the `any` that `Request#json()` produces) so every handler is
 * forced to narrow the payload before reading fields off it.
 */
export async function readJSON(request: Request): Promise<unknown> {
  return request.json().catch(() => null)
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
