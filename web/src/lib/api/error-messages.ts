import { APIError } from '@/lib/api/client'

const API_ERROR_MESSAGE_KEYS = {
  invalid_credentials: 'apiError.invalidCredentials',
  invalid_token: 'apiError.unauthorized',
  account_disabled: 'apiError.accountDisabled',
  unauthorized: 'apiError.unauthorized',
  username_required: 'apiError.usernameRequired',
  username_taken: 'apiError.usernameTaken',
  email_taken: 'apiError.emailTaken',
  current_password_invalid: 'apiError.currentPasswordInvalid',
  password_too_short: 'apiError.passwordTooShort',
  registration_disabled: 'apiError.registrationDisabled',
  setup_required: 'apiError.setupRequired',
  setup_completed: 'apiError.setupCompleted',
  avatar_required: 'apiError.avatarRequired',
  avatar_invalid_type: 'apiError.avatarInvalidType',
  avatar_too_large: 'apiError.avatarTooLarge',
  profile_update_failed: 'apiError.profileUpdateFailed',
  avatar_upload_failed: 'apiError.avatarUploadFailed',
  avatar_update_failed: 'apiError.avatarUploadFailed',
  password_update_failed: 'apiError.passwordUpdateFailed',
  account_delete_failed: 'apiError.accountDeleteFailed',
  account_delete_forbidden: 'apiError.accountDeleteForbidden',
  system_log_stream_unavailable: 'apiError.systemLogStreamFailed',
} as const

type Translate = (key: string) => string
type APIErrorMessageCode = keyof typeof API_ERROR_MESSAGE_KEYS

export function getAPIErrorMessage(
  t: Translate,
  error: unknown,
  fallbackKey = 'apiError.unknown',
): string {
  if (
    error instanceof APIError &&
    error.code !== undefined &&
    error.code !== ''
  ) {
    const key = getAPIErrorMessageKey(error.code)
    if (key) {
      return t(key)
    }
  }

  return t(fallbackKey)
}

function getAPIErrorMessageKey(
  code: string,
): (typeof API_ERROR_MESSAGE_KEYS)[keyof typeof API_ERROR_MESSAGE_KEYS] | null {
  if (!hasAPIErrorMessageKey(code)) {
    return null
  }

  return API_ERROR_MESSAGE_KEYS[code]
}

function hasAPIErrorMessageKey(code: string): code is APIErrorMessageCode {
  return code in API_ERROR_MESSAGE_KEYS
}
