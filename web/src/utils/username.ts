// The 1-32 bound is also spelled out in usernamePattern below. username.spec.ts asserts
// the boundary through this constant, so changing one without the other fails the tests.
export const USERNAME_MAX_LENGTH = 32

/**
 * Mirrors identity.ValidateUsername: ASCII letters/digits plus '.', '_' and '-',
 * 1-32 characters, starting and ending with a letter or digit. The ASCII-only rule
 * keeps homoglyphs, bidi controls and zero-width characters out of login/admin names.
 */
const usernamePattern = /^[A-Za-z0-9](?:[A-Za-z0-9._-]{0,30}[A-Za-z0-9])?$/

export function isValidUsername(value: string): boolean {
  return usernamePattern.test(value.trim())
}
