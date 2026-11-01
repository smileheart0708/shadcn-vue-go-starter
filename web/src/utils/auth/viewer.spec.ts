import { describe, expect, it } from 'vitest'
import type { Viewer } from '@/lib/api/auth'
import { isSameViewer } from '@/utils/auth/viewer'

function createViewer(): Viewer {
  return {
    identity: {
      id: 1,
      username: 'alice',
      email: 'alice@example.com',
      avatarUrl: null,
      status: 'active',
    },
    authorization: {
      role: 'owner',
      capabilities: ['system.settings.read', 'management.users.read'],
    },
  }
}

describe('isSameViewer', () => {
  it('treats a re-parsed payload with identical values as unchanged', () => {
    expect(isSameViewer(createViewer(), createViewer())).toBe(true)
  })

  it('detects a change inside identity or authorization', () => {
    const base = createViewer()

    const changedIdentity = createViewer()
    changedIdentity.identity.username = 'bob'

    const changedAuthorization = createViewer()
    changedAuthorization.authorization.capabilities = ['system.settings.read']

    expect(isSameViewer(base, changedIdentity)).toBe(false)
    expect(isSameViewer(base, changedAuthorization)).toBe(false)
  })

  it('compares null viewer state', () => {
    expect(isSameViewer(null, null)).toBe(true)
    expect(isSameViewer(createViewer(), null)).toBe(false)
    expect(isSameViewer(null, createViewer())).toBe(false)
  })
})
