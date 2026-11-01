import { describe, expect, it } from 'vitest'
import { APIError } from '@/lib/api/client'
import {
  resolveAPIErrorMessage,
  resolveErrorMessageOrInternal,
} from '@/lib/api/error-messages'

// The real translator walks the locale bundle. Echoing the key keeps the assertion on the
// resolved key, which is what regresses when a code is added or renamed.
const translate = (key: string) => key

describe('resolveErrorMessageOrInternal', () => {
  it('uses the code-specific copy for a known backend code', () => {
    const error = new APIError(409, 'username taken', 'username_taken')

    expect(resolveErrorMessageOrInternal(translate, error)).toBe(
      'apiError.username_taken',
    )
  })

  it('falls back to the generic copy for a plain Error', () => {
    expect(resolveErrorMessageOrInternal(translate, new Error('boom'))).toBe(
      'apiError.internal_error',
    )
  })

  it('falls back to the generic copy for a code the frontend does not know', () => {
    const error = new APIError(500, 'boom', 'code_from_a_newer_backend')

    expect(resolveErrorMessageOrInternal(translate, error)).toBe(
      'apiError.internal_error',
    )
  })

  it('falls back to the generic copy when the APIError carries no code', () => {
    const error = new APIError(502, 'bad gateway')

    expect(resolveErrorMessageOrInternal(translate, error)).toBe(
      'apiError.internal_error',
    )
  })

  // Callers that pass their own context-specific copy rely on this staying null; turning it
  // into `internal_error` here would silently replace "login failed" with a server fault.
  it('leaves the raw resolver returning null for an unknown error', () => {
    expect(resolveAPIErrorMessage(translate, new Error('boom'))).toBeNull()
  })
})
