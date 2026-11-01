import { describe, expect, it } from 'vitest'
import { normalizeOptionalEmail } from '@/utils/email'

// The registration, profile and admin dialogs all rely on blank input becoming `null`;
// a regression here would send an empty string and bypass the backend's optional-email
// normalization.
describe('normalizeOptionalEmail', () => {
  it('turns blank input into null', () => {
    expect(normalizeOptionalEmail('')).toBeNull()
    expect(normalizeOptionalEmail('   ')).toBeNull()
  })

  it('trims the value that is sent to the API', () => {
    expect(normalizeOptionalEmail('  alice@example.com  ')).toBe(
      'alice@example.com',
    )
  })
})
