import { describe, expect, it } from 'vitest'
import { isValidEmail, normalizeOptionalEmail } from './email'

describe('isValidEmail', () => {
  // The same cases the Go write boundary pins. A frontend check that accepts what the
  // server rejects only buys a round-trip and a toast.
  it('rejects a bare domain and consecutive dots', () => {
    expect(isValidEmail('a@b')).toBe(false)
    expect(isValidEmail('alice..b@example.com')).toBe(false)
    expect(isValidEmail('alice@example.c')).toBe(false)
  })

  it('accepts a practical address', () => {
    expect(isValidEmail('alice@example.com')).toBe(true)
    expect(isValidEmail('a.b+tag@example.co.uk')).toBe(true)
  })
})

describe('normalizeOptionalEmail', () => {
  it('turns blank input into null', () => {
    expect(normalizeOptionalEmail('   ')).toBeNull()
    expect(normalizeOptionalEmail(' alice@example.com ')).toBe(
      'alice@example.com',
    )
  })
})
