// Email reaches the API from three places (registration form, profile dialog, admin user
// dialog) and every one of them has to turn a blank input into `null` the same way. This
// lives here instead of under `utils/settings/` because it is not a preference: it is the
// single rule for "what the backend should store when the field is left empty".
export function normalizeOptionalEmail(value: string): string | null {
  const email = value.trim()
  return email.length > 0 ? email : null
}
