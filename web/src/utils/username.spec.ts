import { describe, expect, it } from 'vitest'
import { isValidUsername, USERNAME_MAX_LENGTH } from './username'

describe('isValidUsername', () => {
  it('accepts ASCII names with separators and enforces the length bounds', () => {
    expect(isValidUsername('a')).toBe(true)
    expect(isValidUsername('Alice-01')).toBe(true)
    expect(isValidUsername('a.b_c-d')).toBe(true)
    expect(isValidUsername(' a.b_c-d ')).toBe(true)
    expect(isValidUsername('a'.repeat(USERNAME_MAX_LENGTH))).toBe(true)
    expect(isValidUsername('a'.repeat(USERNAME_MAX_LENGTH + 1))).toBe(false)
  })

  // The full rule matrix lives in identity.TestValidateUsername; these pin the shape the
  // two implementations must agree on (ASCII only, no '@', no leading/trailing separator).
  it('rejects non-ASCII, zero-width and separator edges', () => {
    expect(isValidUsername('中文')).toBe(false)
    expect(isValidUsername('ad\u200Bmin')).toBe(false)
    expect(isValidUsername('alice@example.com')).toBe(false)
    expect(isValidUsername('alice bob')).toBe(false)
    expect(isValidUsername('.alice')).toBe(false)
    expect(isValidUsername('alice-')).toBe(false)
  })
})
