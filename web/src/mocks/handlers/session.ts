import { toViewer } from '@/mocks/handlers/permissions'
import {
  MOCK_DEV_ACCESS_TOKEN,
  createMockSession,
  jsonError,
  nowISO,
  state,
  type MockUser,
} from '@/mocks/handlers/state'

// Suppressed once the user logs out explicitly, so the dev session is not resurrected
// by the next refresh.
let autoOwnerSessionAvailable = true

function getCurrentUserFromRequest(request: Request): MockUser | null {
  if (!state.installState.setupCompleted || state.session === null) {
    return null
  }

  const authorization = request.headers.get('Authorization')
  if (authorization !== `Bearer ${state.session.accessToken}`) {
    return null
  }

  return getCurrentUserFromSession()
}

function getCurrentUserFromSession(): MockUser | null {
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

/**
 * Resolves the user a refresh may resume: the live session's user, or the owner fixture
 * while the automatic dev session is still allowed.
 */
export function resolveRefreshUser(): MockUser | null {
  return (
    getCurrentUserFromSession() ??
    (autoOwnerSessionAvailable ? getMockOwner() : null)
  )
}

export function issueSession(
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

export function clearSession(suppressAutoOwnerSession = false) {
  state.session = null
  if (suppressAutoOwnerSession) {
    autoOwnerSessionAvailable = false
  }
}

function getMockOwner(): MockUser | null {
  return (
    state.users.find(
      (entry) =>
        entry.id === state.installState.ownerUserId &&
        entry.role === 'owner' &&
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

export function requireSetupCompleted(): Response | null {
  if (!state.installState.setupCompleted) {
    // 503 matches CodeSetupRequired in internal/httpapi/error_codes.go. Answering 403
    // here made dev-mode error handling take a branch the real server never produces.
    return jsonError(503, 'setup_required', 'Complete setup first.')
  }

  return null
}

// Discriminated union, not `{ error: X | null; user: Y | null }`: callers guard with
// `if (error) return error` and then use `user`, which only narrows to a non-null user
// when the two fields are correlated in the type.
type AuthenticatedResult =
  { error: Response; user: null } | { error: null; user: MockUser }

export function requireAuthenticated(request: Request): AuthenticatedResult {
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
