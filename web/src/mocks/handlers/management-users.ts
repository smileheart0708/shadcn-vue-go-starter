import { http } from 'msw'
import { ROLE_KEY, type ManagedUserAction } from '@/lib/api/contract.generated'
import { CAPABILITY } from '@/lib/auth/capabilities.generated'
import { hasCapability } from '@/mocks/handlers/permissions'
import { clearSession, requireAuthenticated } from '@/mocks/handlers/session'
import {
  emailTaken,
  isRecord,
  jsonError,
  jsonSuccess,
  nowISO,
  readJSON,
  state,
  usernameTaken,
  type MockUser,
} from '@/mocks/handlers/state'

/**
 * Object-level rule, mirroring `identity.CanManageUser`: an actor never acts on itself,
 * only the owner manages other accounts, and nobody manages an owner.
 */
function canManageUser(actor: MockUser, target: MockUser) {
  if (actor.id === target.id) {
    return false
  }

  if (actor.role !== ROLE_KEY.owner) {
    return false
  }

  return target.role !== ROLE_KEY.owner
}

function managedUserActions(actor: MockUser, target: MockUser) {
  if (!canManageUser(actor, target)) {
    return []
  }

  const actions: ManagedUserAction[] = ['update']

  if (target.status === 'active') {
    actions.push('disable')
  }

  if (target.status === 'disabled') {
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

function findTarget(params: { readonly id?: string | readonly string[] }) {
  const targetID = Number.parseInt(String(params.id), 10)
  return state.users.find((entry) => entry.id === targetID) ?? null
}

export const managementUsersHandlers = [
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

    const target = findTarget(params)
    if (target === null) {
      return jsonError(404, 'user_not_found', 'User not found.')
    }
    if (!canManageUser(user, target)) {
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

    const target = findTarget(params)
    if (target === null) {
      return jsonError(404, 'user_not_found', 'User not found.')
    }
    if (!canManageUser(user, target)) {
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

    const target = findTarget(params)
    if (target === null) {
      return jsonError(404, 'user_not_found', 'User not found.')
    }
    if (!canManageUser(user, target)) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    target.status = 'active'
    target.updatedAt = nowISO()
    return jsonSuccess(toManagedUser(target, user))
  }),
]
