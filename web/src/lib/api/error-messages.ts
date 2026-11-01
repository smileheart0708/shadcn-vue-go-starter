import { APIError } from '@/lib/api/client'
import type { MessageSchema } from '@/locales'

/**
 * Backend error codes are the contract: this union is derived from the `apiError`
 * block of the locale files, which `internal/httpapi/error_codes_test.go` keeps in
 * lockstep with the Go registry.
 */
type ApiErrorCode = keyof MessageSchema['apiError']

type ApiErrorMessageKey = `apiError.${ApiErrorCode}`

type Translate = (key: ApiErrorMessageKey) => string

/**
 * One entry per backend code, each holding a literal `t()` call.
 *
 * Deliberately not a `code -> key` lookup table: `@intlify/vue-i18n/no-unused-keys`
 * only sees keys that appear literally inside a `t()` call, so an indirection table
 * would make the whole `apiError.*` block look unused. Inlining the call keeps the
 * keys greppable and the lint rule honest, and `satisfies` still forces every code
 * to have an entry (and rejects unknown ones).
 */
const API_ERROR_MESSAGES = {
  invalid_request: (t: Translate) => t('apiError.invalid_request'),
  invalid_credentials: (t: Translate) => t('apiError.invalid_credentials'),
  invalid_token: (t: Translate) => t('apiError.invalid_token'),
  unauthorized: (t: Translate) => t('apiError.unauthorized'),
  forbidden: (t: Translate) => t('apiError.forbidden'),
  account_disabled: (t: Translate) => t('apiError.account_disabled'),
  registration_disabled: (t: Translate) => t('apiError.registration_disabled'),
  account_delete_forbidden: (t: Translate) =>
    t('apiError.account_delete_forbidden'),
  username_required: (t: Translate) => t('apiError.username_required'),
  username_taken: (t: Translate) => t('apiError.username_taken'),
  email_taken: (t: Translate) => t('apiError.email_taken'),
  password_too_short: (t: Translate) => t('apiError.password_too_short'),
  current_password_invalid: (t: Translate) =>
    t('apiError.current_password_invalid'),
  invalid_id: (t: Translate) => t('apiError.invalid_id'),
  user_not_found: (t: Translate) => t('apiError.user_not_found'),
  avatar_required: (t: Translate) => t('apiError.avatar_required'),
  avatar_invalid_type: (t: Translate) => t('apiError.avatar_invalid_type'),
  avatar_too_large: (t: Translate) => t('apiError.avatar_too_large'),
  setup_required: (t: Translate) => t('apiError.setup_required'),
  setup_completed: (t: Translate) => t('apiError.setup_completed'),
  system_log_stream_unavailable: (t: Translate) =>
    t('apiError.system_log_stream_unavailable'),
  internal_error: (t: Translate) => t('apiError.internal_error'),
} as const satisfies Record<ApiErrorCode, (t: Translate) => string>

/**
 * Resolves a backend error code into localized copy.
 *
 * Returns `null` when the error carries no code the frontend knows about, so the
 * caller supplies its own literal fallback:
 *
 * ```ts
 * resolveAPIErrorMessage(t, error) ?? t('auth.signIn.loginFailed')
 * ```
 */
export function resolveAPIErrorMessage(
  t: Translate,
  error: unknown,
): string | null {
  if (!(error instanceof APIError)) {
    return null
  }

  const code = error.code
  if (code === undefined || !isApiErrorCode(code)) {
    return null
  }

  return API_ERROR_MESSAGES[code](t)
}

/**
 * Like `resolveAPIErrorMessage`, but never returns `null`: an error without a code the
 * frontend knows about falls back to the generic internal-error copy.
 *
 * This is the shape most `catch` blocks want — "show the reason, or a safe generic
 * message". Reach for `resolveAPIErrorMessage` directly only when the caller has
 * context-specific copy of its own (a failed sign-in, a failed save) and must not fall
 * back to `internal_error`.
 */
export function resolveErrorMessageOrInternal(
  t: Translate,
  error: unknown,
): string {
  return resolveAPIErrorMessage(t, error) ?? t('apiError.internal_error')
}

function isApiErrorCode(code: string): code is ApiErrorCode {
  return Object.prototype.hasOwnProperty.call(API_ERROR_MESSAGES, code)
}
