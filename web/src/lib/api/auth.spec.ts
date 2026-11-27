import { afterEach, describe, expect, it, vi } from 'vitest'
import { refreshSession } from './auth'

const { post, refreshState } = vi.hoisted(() => {
  const state = {
    inFlight: 0,
    maxInFlight: 0,
  }

  return {
    refreshState: state,
    post: vi.fn(() => {
      state.inFlight += 1
      state.maxInFlight = Math.max(state.maxInFlight, state.inFlight)

      return {
        json: async () => {
          await new Promise((resolve) => setTimeout(resolve, 5))
          state.inFlight -= 1

          return {
            success: true,
            data: {
              accessToken: 'access-token',
              tokenType: 'Bearer',
              expiresAt: '2026-11-27T12:00:00Z',
              viewer: {
                identity: {
                  id: 1,
                  username: 'owner',
                  email: null,
                  avatarUrl: null,
                  status: 'active',
                },
                authorization: {
                  role: 'owner',
                  capabilities: [],
                },
              },
            },
          }
        },
      }
    }),
  }
})

vi.mock('@/lib/api/client', () => ({
  authApi: {},
  baseApi: { post },
  normalizeAPIError: (error: unknown) => {
    throw error instanceof Error ? error : new Error(String(error))
  },
}))

function createSerialLockManager() {
  let tail: Promise<unknown> = Promise.resolve()

  return {
    request<T>(_name: string, callback: () => Promise<T>): Promise<T> {
      const run = tail.then(callback, callback)
      tail = run.catch(() => undefined)
      return run
    },
  }
}

afterEach(() => {
  Reflect.deleteProperty(navigator, 'locks')
  refreshState.inFlight = 0
  refreshState.maxInFlight = 0
})

describe('refreshSession', () => {
  // The refresh cookie is shared across tabs. Without the origin-wide lock, two tabs
  // send the same token; the server treats the loser as token reuse and revokes the
  // whole session. Capping concurrency at one is the observable contract here.
  it('serializes concurrent refreshes through the cross-tab lock', async () => {
    Object.defineProperty(navigator, 'locks', {
      configurable: true,
      value: createSerialLockManager(),
    })

    const [first, second] = await Promise.all([
      refreshSession(),
      refreshSession(),
    ])

    expect(first.accessToken).toBe('access-token')
    expect(second.accessToken).toBe('access-token')
    expect(refreshState.maxInFlight).toBe(1)
  })
})
