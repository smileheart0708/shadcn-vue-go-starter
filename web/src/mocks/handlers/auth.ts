import { http } from 'msw'
import { toViewer } from '@/mocks/handlers/permissions'
import {
  clearSession,
  issueSession,
  requireAuthenticated,
  requireSetupCompleted,
  resolveRefreshUser,
} from '@/mocks/handlers/session'
import {
  emailTaken,
  getPublicAuthConfig,
  isRecord,
  jsonError,
  jsonSuccess,
  nowISO,
  readJSON,
  state,
  usernameTaken,
  type MockUser,
} from '@/mocks/handlers/state'

export const authHandlers = [
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
    const user = resolveRefreshUser()
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
]
