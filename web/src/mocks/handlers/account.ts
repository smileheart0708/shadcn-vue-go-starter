import { http } from 'msw'
import { CAPABILITY } from '@/lib/auth/capabilities.generated'
import { hasCapability, toViewer } from '@/mocks/handlers/permissions'
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
} from '@/mocks/handlers/state'

export const accountHandlers = [
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
]
