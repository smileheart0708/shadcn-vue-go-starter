import { http } from 'msw'
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
    if (state.installState.phase === 'completed') {
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

    if (state.installState.phase === 'database_required') {
      if (!isRecord(payload.database)) {
        return jsonError(
          400,
          'database_input_required',
          'Choose a database before completing setup.',
        )
      }
      const driver = payload.database.driver
      if (driver !== 'sqlite' && driver !== 'postgresql') {
        return jsonError(
          400,
          'database_config_invalid',
          'Invalid database configuration.',
        )
      }
      state.installState.databaseDriver = driver
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
      phase: 'completed',
      databaseDriver: state.installState.databaseDriver ?? 'sqlite',
      ownerUserId: owner.id,
      completedAt: createdAt,
    }
    state.accountPolicies = {
      publicRegistrationEnabled: false,
      selfServiceAccountDeletionEnabled: false,
    }

    return jsonSuccess(state.installState, { status: 201 })
  }),
]
