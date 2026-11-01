import { http } from 'msw'
import { CAPABILITY } from '@/lib/auth/capabilities.generated'
import { hasCapability } from '@/mocks/handlers/permissions'
import { requireAuthenticated } from '@/mocks/handlers/session'
import {
  isRecord,
  jsonError,
  jsonSuccess,
  readJSON,
  state,
} from '@/mocks/handlers/state'

export const systemSettingsHandlers = [
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
]
