import imageCompression from 'browser-image-compression'
import {
  isSupportedAvatarFileType,
  MAX_AVATAR_FILE_SIZE_BYTES,
} from '@/lib/avatar'

export type AvatarValidationFailure = 'unsupported-type' | 'file-too-large'

export type AvatarValidationResult =
  { ok: true } | { ok: false; failure: AvatarValidationFailure }

export function validateAvatarFile(file: File): AvatarValidationResult {
  if (!isSupportedAvatarFileType(file.type)) {
    return { ok: false, failure: 'unsupported-type' }
  }

  if (file.size > MAX_AVATAR_FILE_SIZE_BYTES) {
    return { ok: false, failure: 'file-too-large' }
  }

  return { ok: true }
}

export async function compressAvatarFile(file: File): Promise<File> {
  const compressedAvatar = await imageCompression(file, {
    maxSizeMB: 0.4,
    maxWidthOrHeight: 512,
    useWebWorker: true,
    fileType: file.type,
    initialQuality: 0.8,
  })

  // browser-image-compression can resolve to a plain Blob, so re-wrap it to preserve the
  // original filename for the multipart upload.
  return compressedAvatar instanceof File
    ? compressedAvatar
    : new File([compressedAvatar], file.name, { type: file.type })
}
