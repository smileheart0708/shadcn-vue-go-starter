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

  it('detects a change in every identity field', () => {
    const changedId = createViewer()
    changedId.identity.id = 2

    const changedUsername = createViewer()
    changedUsername.identity.username = 'bob'

    const clearedEmail = createViewer()
    clearedEmail.identity.email = null

    const addedAvatar = createViewer()
    addedAvatar.identity.avatarUrl = 'https://example.com/avatar.png'

    const base = createViewer()

    expect(isSameViewer(base, changedId)).toBe(false)
    expect(isSameViewer(base, changedUsername)).toBe(false)
    expect(isSameViewer(base, clearedEmail)).toBe(false)
    expect(isSameViewer(base, addedAvatar)).toBe(false)
  })

  it('detects identity status and authorization changes', () => {
    const changedStatus = createViewer()
    changedStatus.identity.status = 'disabled'

    const changedRole = createViewer()
    changedRole.authorization.role = 'user'

    const removedCapability = createViewer()
    removedCapability.authorization.capabilities = ['system.settings.read']

    const reorderedCapabilities = createViewer()
    reorderedCapabilities.authorization.capabilities = [
      'management.users.read',
      'system.settings.read',
    ]

    const base = createViewer()

    expect(isSameViewer(base, changedStatus)).toBe(false)
    expect(isSameViewer(base, changedRole)).toBe(false)
    expect(isSameViewer(base, removedCapability)).toBe(false)
    expect(isSameViewer(base, reorderedCapabilities)).toBe(false)
  })

  it('compares null viewer state', () => {
    expect(isSameViewer(null, null)).toBe(true)
    expect(isSameViewer(createViewer(), null)).toBe(false)
    expect(isSameViewer(null, createViewer())).toBe(false)
  })
})
