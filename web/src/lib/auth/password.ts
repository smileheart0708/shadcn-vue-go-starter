/**
 * Mirrors `auth.MinPasswordLength` in `internal/auth/password.go`.
 *
 * The server stays the only enforcement point; this constant exists so the native
 * `minlength` attribute cannot drift between the setup, register and change-password
 * forms. When the Go constant changes, update this value and the
 * `apiError.password_too_short` copy, which spells the number out.
 *
 * The two sides do not count the same unit: `minlength` counts UTF-16 code units,
 * `auth.passwordTooShort` counts code points. A code-unit count is never smaller than a
 * code-point count, so the native bound can only be looser than the server. Do not make
 * the client stricter (for example by counting bytes), or a password the server accepts
 * becomes unsubmittable with a browser message the UI cannot translate.
 */
export const MIN_PASSWORD_LENGTH = 8

/**
 * The character-count rule, for the MSW handlers that stand in for the Go server.
 *
 * `Array.from` iterates code points like `utf8.RuneCountInString`; `value.length` would
 * count astral characters twice and reject passwords the real server accepts. Keeping the
 * handlers on this function also stops a second copy of the number from appearing in
 * `src/mocks`.
 */
export function meetsMinPasswordLength(value: string): boolean {
  return Array.from(value.trim()).length >= MIN_PASSWORD_LENGTH
}
