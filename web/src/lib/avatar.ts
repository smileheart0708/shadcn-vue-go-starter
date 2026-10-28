export function getAvatarFallbackText(name: string | null | undefined): string {
  const normalizedName = name?.trim()

  if (normalizedName === undefined || normalizedName === '') {
    return '??'
  }

  return Array.from(normalizedName).slice(0, 2).join('').toUpperCase()
}

/**
 * Single source of truth for avatar upload constraints. The file input's `accept`
 * string and the runtime type guard are both derived from this list — add new formats
 * here only, and mirror the change in the backend's avatar validation.
 */
export const SUPPORTED_AVATAR_FILE_TYPES = [
  'image/jpeg',
  'image/png',
  'image/webp',
] as const

export type SupportedAvatarFileType =
  (typeof SUPPORTED_AVATAR_FILE_TYPES)[number]

export const AVATAR_ACCEPTED_FILE_TYPES = SUPPORTED_AVATAR_FILE_TYPES.join(',')

export const MAX_AVATAR_FILE_SIZE_BYTES = 5 * 1024 * 1024

export function isSupportedAvatarFileType(
  fileType: string,
): fileType is SupportedAvatarFileType {
  return SUPPORTED_AVATAR_FILE_TYPES.some((candidate) => candidate === fileType)
}
