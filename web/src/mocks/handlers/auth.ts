import { http, HttpResponse } from 'msw'
import type { RoleKey, ViewerIdentity } from '@/lib/api/auth'
import { CAPABILITY, type Capability } from '@/lib/auth/capabilities.generated'
import { ROLE_KEY } from '@/lib/auth/roles'

/**
 * In-memory backend stand-in for `pnpm dev` with `VITE_API_MOCKING=true`.
 *
 * Domain types are imported from `lib/api/auth` rather than redeclared so that a schema
 * change breaks this file at compile time instead of silently drifting from the client
 * contract it is supposed to emulate. The role matrix below mirrors
 * `internal/authorization/policy.go`; `satisfies Record<RoleKey, ...>` makes a new
 * backend role a typecheck failure here until the mock is taught about it.
 */

type UserStatus = ViewerIdentity['status']

interface MockUser {
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
const MOCK_DEV_ACCESS_TOKEN = 'mock-owner-access-token'
const MOCK_FIXTURE_CREATED_AT = '2026-01-01T00:00:00.000Z'
const MOCK_FIXTURE_UPDATED_AT = '2026-01-02T00:00:00.000Z'

function nowISO() {
  return new Date().toISOString()
}

function createMockSession(userId: number, accessToken: string): MockSession {
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

const state = createInitialState()
let autoOwnerSessionAvailable = true

const baseSystemLogs = [
  createSystemLogEntry(1, 'INFO', 'mock', 'Mock system log stream connected.'),
  createSystemLogEntry(2, 'INFO', 'auth', 'Mock owner session initialized.'),
  createSystemLogEntry(3, 'DEBUG', 'setup', 'Mock install state is completed.'),
]

function createSystemLogEntry(
  id: number,
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR',
  source: string,
  text: string,
) {
  return {
    id,
    timestamp: Math.floor(Date.now() / 1000),
    level,
    source,
    message: text,
    text,
  }
}

function jsonSuccess(data: unknown, init?: ResponseInit) {
  return HttpResponse.json({ success: true, data }, init)
}

function jsonError(status: number, code: string, message: string) {
  return HttpResponse.json(
    { success: false, error: { code, message } },
    { status },
  )
}

function getPublicAuthConfig() {
  return {
    registrationEnabled: state.accountPolicies.publicRegistrationEnabled,
  }
}

function getCurrentUserFromRequest(request: Request) {
  if (!state.installState.setupCompleted || state.session === null) {
    return null
  }

  const authorization = request.headers.get('Authorization')
  if (authorization !== `Bearer ${state.session.accessToken}`) {
    return null
  }

  const user =
    state.users.find((entry) => entry.id === state.session?.userId) ?? null
  if (user?.status !== 'active') {
    return null
  }

  return user
}

function getCurrentUserFromSession() {
  if (state.session === null) {
    return null
  }

  const user =
    state.users.find((entry) => entry.id === state.session?.userId) ?? null
  if (user?.status !== 'active') {
    return null
  }

  return user
}

const ROLE_CAPABILITIES = {
  [ROLE_KEY.owner]: [
    CAPABILITY.systemSettingsRead,
    CAPABILITY.systemSettingsUpdate,
    CAPABILITY.managementUsersRead,
    CAPABILITY.managementUsersCreate,
    CAPABILITY.managementUsersUpdate,
    CAPABILITY.managementUsersDisable,
    CAPABILITY.managementUsersEnable,
    CAPABILITY.managementSystemLogsRead,
  ],
  [ROLE_KEY.user]: [],
} as const satisfies Record<RoleKey, readonly Capability[]>

function capabilitiesFor(user: MockUser): Capability[] {
  const capabilities = new Set<Capability>(ROLE_CAPABILITIES[user.role])

  // Mirrors the server rule: self-deletion is policy-driven and never granted to
  // the owner, who could otherwise delete the only administrative account.
  if (
    state.accountPolicies.selfServiceAccountDeletionEnabled &&
    user.role !== ROLE_KEY.owner
  ) {
    capabilities.add(CAPABILITY.accountDeleteSelf)
  }

  return [...capabilities].sort()
}

function toViewer(user: MockUser) {
  return {
    identity: {
      id: user.id,
      username: user.username,
      email: user.email,
      avatarUrl: user.avatarUrl,
      status: user.status,
    },
    authorization: {
      role: user.role,
      capabilities: capabilitiesFor(user),
    },
  }
}

function issueSession(
  user: MockUser,
  accessToken = `mock-access-${String(user.id)}-${String(Date.now())}`,
) {
  user.lastActiveAt = nowISO()
  state.session = createMockSession(user.id, accessToken)

  return {
    accessToken: state.session.accessToken,
    tokenType: 'Bearer',
    expiresAt: state.session.expiresAt,
    viewer: toViewer(user),
  }
}

function clearSession(suppressAutoOwnerSession = false) {
  state.session = null
  if (suppressAutoOwnerSession) {
    autoOwnerSessionAvailable = false
  }
}

function getMockOwner() {
  return (
    state.users.find(
      (entry) =>
        entry.id === state.installState.ownerUserId &&
        entry.role === ROLE_KEY.owner &&
        entry.status === 'active',
    ) ?? null
  )
}

export function ensureMockOwnerSession() {
  const owner = getMockOwner()
  if (owner === null) {
    throw new Error('Mock owner fixture is unavailable.')
  }

  autoOwnerSessionAvailable = true
  const sessionResponse = issueSession(owner, MOCK_DEV_ACCESS_TOKEN)
  return sessionResponse.accessToken
}

function usernameTaken(username: string, excludeUserID?: number) {
  return state.users.some(
    (user) =>
      user.username.toLowerCase() === username.toLowerCase() &&
      user.id !== excludeUserID,
  )
}

function emailTaken(email: string | null, excludeUserID?: number) {
  if (email === null || email === '') {
    return false
  }

  return state.users.some(
    (user) =>
      user.email?.toLowerCase() === email.toLowerCase() &&
      user.id !== excludeUserID,
  )
}

function requireSetupCompleted() {
  if (!state.installState.setupCompleted) {
    return jsonError(403, 'setup_required', 'Complete setup first.')
  }

  return null
}

// Discriminated union, not `{ error: X | null; user: Y | null }`: callers guard with
// `if (error) return error` and then use `user`, which only narrows to a non-null user
// when the two fields are correlated in the type.
type AuthenticatedResult =
  { error: Response; user: null } | { error: null; user: MockUser }

function requireAuthenticated(request: Request): AuthenticatedResult {
  const setupResponse = requireSetupCompleted()
  if (setupResponse) {
    return { error: setupResponse, user: null }
  }

  const user = getCurrentUserFromRequest(request)
  if (user === null) {
    return {
      error: jsonError(401, 'unauthorized', 'Authentication required.'),
      user: null,
    }
  }

  return { error: null, user }
}

function hasCapability(user: MockUser, capability: Capability) {
  return capabilitiesFor(user).includes(capability)
}

function canManageUser(
  actor: MockUser,
  target: MockUser,
  _action: 'update' | 'disable' | 'enable',
) {
  if (actor.id === target.id) {
    return false
  }

  if (actor.role !== ROLE_KEY.owner) {
    return false
  }

  return target.role !== ROLE_KEY.owner
}

function managedUserActions(actor: MockUser, target: MockUser) {
  const actions: ('update' | 'disable' | 'enable')[] = []

  if (canManageUser(actor, target, 'update')) {
    actions.push('update')
  }

  if (target.status === 'active' && canManageUser(actor, target, 'disable')) {
    actions.push('disable')
  }

  if (target.status === 'disabled' && canManageUser(actor, target, 'enable')) {
    actions.push('enable')
  }

  return actions
}

function toManagedUser(user: MockUser, actor: MockUser) {
  return {
    id: user.id,
    username: user.username,
    email: user.email,
    avatarUrl: user.avatarUrl,
    role: user.role,
    status: user.status,
    lastActiveAt: user.lastActiveAt,
    createdAt: user.createdAt,
    updatedAt: user.updatedAt,
    actions: managedUserActions(actor, user),
  }
}

function readPositiveInt(url: URL, key: string, fallback: number) {
  const raw = url.searchParams.get(key)
  const parsed = raw === null ? Number.NaN : Number.parseInt(raw, 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

function formatSystemLogEvent(entry: ReturnType<typeof createSystemLogEntry>) {
  return `event: log\nid: ${String(entry.id)}\ndata: ${JSON.stringify(entry)}\n\n`
}

function buildSystemLogStreamResponse(rawTail: string | null) {
  const encoder = new TextEncoder()
  let intervalId: ReturnType<typeof setInterval> | null = null
  const replayEntries = selectSystemLogReplayEntries(rawTail)

  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      for (const entry of replayEntries) {
        controller.enqueue(encoder.encode(formatSystemLogEvent(entry)))
      }

      intervalId = setInterval(() => {
        const entry = createSystemLogEntry(
          state.nextLogId++,
          'INFO',
          'mock',
          'Mock system log placeholder heartbeat.',
        )
        controller.enqueue(encoder.encode(formatSystemLogEvent(entry)))
      }, 5_000)
    },
    cancel() {
      if (intervalId !== null) {
        clearInterval(intervalId)
      }
    },
  })

  return new HttpResponse(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      Connection: 'keep-alive',
    },
  })
}

function selectSystemLogReplayEntries(rawTail: string | null) {
  const normalizedTail = rawTail?.trim().toLowerCase()
  if (
    normalizedTail === undefined ||
    normalizedTail === '' ||
    normalizedTail === 'all'
  ) {
    return baseSystemLogs
  }

  const tail = Number(normalizedTail)
  if (tail === 100 || tail === 200 || tail === 500 || tail === 1000) {
    return baseSystemLogs.slice(-tail)
  }

  return baseSystemLogs
}

/**
 * Returns `unknown` (not the `any` that `Request#json()` produces) so every handler is
 * forced to narrow the payload before reading fields off it.
 */
async function readJSON(request: Request): Promise<unknown> {
  return request.json().catch(() => null)
}

export const authHandlers = [
  http.get('/api/install/state', () => jsonSuccess(state.installState)),

  http.post('/api/install/setup', async ({ request }) => {
    if (state.installState.setupCompleted) {
      return jsonError(
        409,
        'setup_completed',
        'Setup has already been completed.',
      )
    }

    const payload = await readJSON(request)
    if (
      !isRecord(payload) ||
      typeof payload.username !== 'string' ||
      typeof payload.password !== 'string'
    ) {
      return jsonError(400, 'invalid_request', 'Invalid setup payload.')
    }

    const username = payload.username.trim()
    if (username === '') {
      return jsonError(400, 'username_required', 'Username is required.')
    }
    if (payload.password.trim().length < 8) {
      return jsonError(
        400,
        'password_too_short',
        'Password must be at least 8 characters.',
      )
    }
    if (usernameTaken(username)) {
      return jsonError(409, 'username_taken', 'Username is already in use.')
    }

    const createdAt = nowISO()
    const owner: MockUser = {
      id: state.nextUserId++,
      username,
      email: null,
      avatarUrl: null,
      role: 'owner',
      status: 'active',
      password: payload.password,
      lastActiveAt: null,
      createdAt,
      updatedAt: createdAt,
    }

    state.users = [owner]
    state.installState = {
      setupCompleted: true,
      ownerUserId: owner.id,
      completedAt: createdAt,
    }
    state.accountPolicies = {
      publicRegistrationEnabled: false,
      selfServiceAccountDeletionEnabled: false,
    }

    const sessionResponse = issueSession(owner)
    return jsonSuccess(sessionResponse, { status: 201 })
  }),

  http.get('/api/auth/public-config', () => jsonSuccess(getPublicAuthConfig())),

  http.post('/api/auth/login', async ({ request }) => {
    const setupResponse = requireSetupCompleted()
    if (setupResponse) {
      return setupResponse
    }

    const payload = await readJSON(request)
    if (
      !isRecord(payload) ||
      typeof payload.identifier !== 'string' ||
      typeof payload.password !== 'string'
    ) {
      return jsonError(400, 'invalid_request', 'Invalid login payload.')
    }

    const user =
      state.users.find(
        (entry) =>
          entry.username === payload.identifier ||
          entry.email === payload.identifier,
      ) ?? null
    if (user?.password !== payload.password) {
      return jsonError(401, 'invalid_credentials', 'Invalid credentials.')
    }
    if (user.status !== 'active') {
      return jsonError(403, 'account_disabled', 'Account is disabled.')
    }

    const sessionResponse = issueSession(user)
    return jsonSuccess(sessionResponse)
  }),

  http.post('/api/auth/register', async ({ request }) => {
    const setupResponse = requireSetupCompleted()
    if (setupResponse) {
      return setupResponse
    }

    if (!state.accountPolicies.publicRegistrationEnabled) {
      return jsonError(
        403,
        'registration_disabled',
        'Registration is disabled.',
      )
    }

    const payload = await readJSON(request)
    if (
      !isRecord(payload) ||
      typeof payload.username !== 'string' ||
      typeof payload.password !== 'string'
    ) {
      return jsonError(400, 'invalid_request', 'Invalid registration payload.')
    }

    const username = payload.username.trim()
    const email =
      typeof payload.email === 'string' ? payload.email.trim() || null : null
    if (username === '') {
      return jsonError(400, 'username_required', 'Username is required.')
    }
    if (payload.password.trim().length < 8) {
      return jsonError(
        400,
        'password_too_short',
        'Password must be at least 8 characters.',
      )
    }
    if (usernameTaken(username)) {
      return jsonError(409, 'username_taken', 'Username is already in use.')
    }
    if (emailTaken(email)) {
      return jsonError(409, 'email_taken', 'Email is already in use.')
    }

    const createdAt = nowISO()
    const user: MockUser = {
      id: state.nextUserId++,
      username,
      email,
      avatarUrl: null,
      role: 'user',
      status: 'active',
      password: payload.password,
      lastActiveAt: null,
      createdAt,
      updatedAt: createdAt,
    }

    state.users = [...state.users, user]
    const sessionResponse = issueSession(user)
    return jsonSuccess(sessionResponse, { status: 201 })
  }),

  http.post('/api/auth/refresh', () => {
    const user =
      getCurrentUserFromSession() ??
      (autoOwnerSessionAvailable ? getMockOwner() : null)
    if (user === null) {
      clearSession()
      return jsonError(401, 'invalid_token', 'Refresh token is invalid.')
    }

    const sessionResponse = issueSession(user)
    return jsonSuccess(sessionResponse)
  }),

  http.post('/api/auth/logout', () => {
    clearSession(true)
    return jsonSuccess({ loggedOut: true })
  }),

  http.get('/api/auth/me', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    return jsonSuccess(toViewer(user))
  }),

  http.patch('/api/account/profile', async ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    const payload = await readJSON(request)
    if (!isRecord(payload) || typeof payload.username !== 'string') {
      return jsonError(400, 'invalid_request', 'Invalid profile payload.')
    }

    const username = payload.username.trim()
    const email =
      typeof payload.email === 'string' ? payload.email.trim() || null : null
    if (username === '') {
      return jsonError(400, 'username_required', 'Username is required.')
    }
    if (usernameTaken(username, user.id)) {
      return jsonError(409, 'username_taken', 'Username is already in use.')
    }
    if (emailTaken(email, user.id)) {
      return jsonError(409, 'email_taken', 'Email is already in use.')
    }

    user.username = username
    user.email = email
    user.updatedAt = nowISO()
    return jsonSuccess(toViewer(user))
  }),

  http.post('/api/account/avatar', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    user.avatarUrl = `/mock/avatar-${String(user.id)}.png?v=${String(Date.now())}`
    user.updatedAt = nowISO()
    return jsonSuccess(toViewer(user))
  }),

  http.post('/api/account/password', async ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    const payload = await readJSON(request)
    if (
      !isRecord(payload) ||
      typeof payload.currentPassword !== 'string' ||
      typeof payload.newPassword !== 'string'
    ) {
      return jsonError(400, 'invalid_request', 'Invalid password payload.')
    }
    if (payload.currentPassword !== user.password) {
      return jsonError(
        400,
        'current_password_invalid',
        'Current password is invalid.',
      )
    }
    if (payload.newPassword.trim().length < 8) {
      return jsonError(
        400,
        'password_too_short',
        'Password must be at least 8 characters.',
      )
    }

    user.password = payload.newPassword
    user.updatedAt = nowISO()
    clearSession(true)
    return jsonSuccess({ passwordChanged: true })
  }),

  http.delete('/api/account', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, CAPABILITY.accountDeleteSelf)) {
      return jsonError(
        403,
        'account_delete_forbidden',
        'This account cannot delete itself.',
      )
    }

    state.users = state.users.filter((entry) => entry.id !== user.id)
    clearSession(true)
    return jsonSuccess({ deleted: true })
  }),

  http.get('/api/system/settings', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, CAPABILITY.systemSettingsRead)) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    return jsonSuccess(state.accountPolicies)
  }),

  http.patch('/api/system/settings', async ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, CAPABILITY.systemSettingsUpdate)) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    const payload = await readJSON(request)
    if (!isRecord(payload)) {
      return jsonError(400, 'invalid_request', 'Invalid settings payload.')
    }

    if (typeof payload.publicRegistrationEnabled === 'boolean') {
      state.accountPolicies.publicRegistrationEnabled =
        payload.publicRegistrationEnabled
    }
    if (typeof payload.selfServiceAccountDeletionEnabled === 'boolean') {
      state.accountPolicies.selfServiceAccountDeletionEnabled =
        payload.selfServiceAccountDeletionEnabled
    }

    return jsonSuccess(state.accountPolicies)
  }),

  http.get('/api/management/users', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, CAPABILITY.managementUsersRead)) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    const url = new URL(request.url)
    const query = (url.searchParams.get('q') ?? '').trim().toLowerCase()
    const status = url.searchParams.get('status')
    const page = readPositiveInt(url, 'page', 1)
    const pageSize = readPositiveInt(url, 'pageSize', 20)

    const filtered = state.users.filter((entry) => {
      if (
        query !== '' &&
        ![entry.username, entry.email ?? ''].some((value) =>
          value.toLowerCase().includes(query),
        )
      ) {
        return false
      }
      if (status !== null && status !== '' && status !== entry.status) {
        return false
      }
      return true
    })

    const start = (page - 1) * pageSize
    const items = filtered
      .slice(start, start + pageSize)
      .map((entry) => toManagedUser(entry, user))

    return jsonSuccess({
      items,
      page,
      pageSize,
      total: filtered.length,
    })
  }),

  http.post('/api/management/users', async ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, CAPABILITY.managementUsersCreate)) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    const payload = await readJSON(request)
    if (
      !isRecord(payload) ||
      typeof payload.username !== 'string' ||
      typeof payload.password !== 'string'
    ) {
      return jsonError(400, 'invalid_request', 'Invalid user payload.')
    }

    const username = payload.username.trim()
    const email =
      typeof payload.email === 'string' ? payload.email.trim() || null : null
    if (username === '') {
      return jsonError(400, 'username_required', 'Username is required.')
    }
    if (payload.password.trim().length < 8) {
      return jsonError(
        400,
        'password_too_short',
        'Password must be at least 8 characters.',
      )
    }
    if (usernameTaken(username)) {
      return jsonError(409, 'username_taken', 'Username is already in use.')
    }
    if (emailTaken(email)) {
      return jsonError(409, 'email_taken', 'Email is already in use.')
    }

    const createdAt = nowISO()
    const createdUser: MockUser = {
      id: state.nextUserId++,
      username,
      email,
      avatarUrl: null,
      role: 'user',
      status: 'active',
      password: payload.password,
      lastActiveAt: null,
      createdAt,
      updatedAt: createdAt,
    }

    state.users = [...state.users, createdUser]
    return jsonSuccess(toManagedUser(createdUser, user), { status: 201 })
  }),

  http.patch('/api/management/users/:id', async ({ request, params }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    const targetID = Number.parseInt(String(params.id), 10)
    const target = state.users.find((entry) => entry.id === targetID) ?? null
    if (target === null) {
      return jsonError(404, 'user_not_found', 'User not found.')
    }
    if (!canManageUser(user, target, 'update')) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    const payload = await readJSON(request)
    if (!isRecord(payload) || typeof payload.username !== 'string') {
      return jsonError(400, 'invalid_request', 'Invalid user payload.')
    }

    const username = payload.username.trim()
    const email =
      typeof payload.email === 'string' ? payload.email.trim() || null : null
    if (username === '') {
      return jsonError(400, 'username_required', 'Username is required.')
    }
    if (usernameTaken(username, target.id)) {
      return jsonError(409, 'username_taken', 'Username is already in use.')
    }
    if (emailTaken(email, target.id)) {
      return jsonError(409, 'email_taken', 'Email is already in use.')
    }

    target.username = username
    target.email = email
    target.updatedAt = nowISO()

    return jsonSuccess(toManagedUser(target, user))
  }),

  http.post('/api/management/users/:id/disable', ({ request, params }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    const targetID = Number.parseInt(String(params.id), 10)
    const target = state.users.find((entry) => entry.id === targetID) ?? null
    if (target === null) {
      return jsonError(404, 'user_not_found', 'User not found.')
    }
    if (!canManageUser(user, target, 'disable')) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    target.status = 'disabled'
    target.updatedAt = nowISO()
    if (state.session?.userId === target.id) {
      clearSession()
    }
    return jsonSuccess(toManagedUser(target, user))
  }),

  http.post('/api/management/users/:id/enable', ({ request, params }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }

    const targetID = Number.parseInt(String(params.id), 10)
    const target = state.users.find((entry) => entry.id === targetID) ?? null
    if (target === null) {
      return jsonError(404, 'user_not_found', 'User not found.')
    }
    if (!canManageUser(user, target, 'enable')) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    target.status = 'active'
    target.updatedAt = nowISO()
    return jsonSuccess(toManagedUser(target, user))
  }),

  http.get('/api/management/system-logs/stream', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, CAPABILITY.managementSystemLogsRead)) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    const url = new URL(request.url)
    return buildSystemLogStreamResponse(url.searchParams.get('tail'))
  }),
]

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
