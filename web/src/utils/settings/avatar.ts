import {
  isSupportedAvatarFileType,
  MAX_AVATAR_FILE_SIZE_BYTES,
} from '@/lib/avatar'

/**
 * Avatars render at most at 80 CSS px (`EditProfileDialog`), so 256 px covers a
 * 3x display density. Anything larger is wasted bytes on every request.
 */
const AVATAR_OUTPUT_SIZE = 256
const AVATAR_OUTPUT_QUALITY = 0.85

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

function hasTransparentPixels(
  context: CanvasRenderingContext2D,
  size: number,
): boolean {
  const { data } = context.getImageData(0, 0, size, size)

  for (let index = 3; index < data.length; index += 4) {
    if (data[index] !== 255) {
      return true
    }
  }

  return false
}

export async function compressAvatarFile(file: File): Promise<File> {
  const bitmap = await createImageBitmap(file, {
    imageOrientation: 'from-image',
  })
  const sourceSize = Math.min(bitmap.width, bitmap.height)
  const sourceX = (bitmap.width - sourceSize) / 2
  const sourceY = (bitmap.height - sourceSize) / 2

  const canvas = document.createElement('canvas')
  canvas.width = AVATAR_OUTPUT_SIZE
  canvas.height = AVATAR_OUTPUT_SIZE
  const context = canvas.getContext('2d')
  if (context === null) {
    bitmap.close()
    throw new Error('avatar canvas context unavailable')
  }

  context.imageSmoothingQuality = 'high'
  context.drawImage(
    bitmap,
    sourceX,
    sourceY,
    sourceSize,
    sourceSize,
    0,
    0,
    AVATAR_OUTPUT_SIZE,
    AVATAR_OUTPUT_SIZE,
  )
  bitmap.close()

  // JPEG has no alpha channel, so a transparent avatar has to stay PNG or its
  // backdrop would be lost. Checking the 256x256 result is cheap and also
  // catches the feathered edges of a logo.
  const mimeType = hasTransparentPixels(context, AVATAR_OUTPUT_SIZE)
    ? 'image/png'
    : 'image/jpeg'

  const blob = await new Promise<Blob | null>((resolve) => {
    canvas.toBlob(resolve, mimeType, AVATAR_OUTPUT_QUALITY)
  })
  if (blob === null) {
    throw new Error('avatar encoding failed')
  }

  // `blob.type` is the only source of truth: a browser that cannot encode the
  // requested type silently falls back to PNG instead of failing.
  const extension = blob.type === 'image/png' ? 'png' : 'jpg'
  return new File([blob], `avatar.${extension}`, { type: blob.type })
}
