import { http } from 'msw'
import { issueSession } from '@/mocks/handlers/session'
import {
  isRecord,
  jsonError,
  jsonSuccess,
  nowISO,
  readJSON,
  state,
  usernameTaken,
  type MockUser,
} from '@/mocks/handlers/state'

export const installHandlers = [
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
]
