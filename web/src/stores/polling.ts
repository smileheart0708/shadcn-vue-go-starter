import { useStorage } from '@vueuse/core'
import { computed } from 'vue'
import { defineStore } from 'pinia'
import { z } from 'zod'
import { parseExternalValueOrDefault } from '@/lib/external-input'

const CURRENT_USER_INTERVAL_STORAGE_KEY =
  'app.polling.current-user-interval-seconds'

export const POLLING_INTERVAL_MIN_SECONDS = 5
export const POLLING_INTERVAL_MAX_SECONDS = 60
export const POLLING_INTERVAL_DEFAULT_SECONDS = 10
// Deliberately NOT `z.coerce.number()`: coercion maps both `null` and `''` to 0, which
// then survives the `Number.isFinite` guard below and gets clamped to the *minimum*
// (5s) instead of falling back to the default (10s) — silently doubling the poll rate
// whenever localStorage is cleared or corrupted.
const storedPollingIntervalSecondsSchema = z
  .string()
  .trim()
  .min(1)
  .transform((value) => Number(value))
  .refine(Number.isFinite)
  .transform(normalizePollingIntervalSeconds)

export function normalizePollingIntervalSeconds(value: number) {
  if (!Number.isFinite(value)) {
    return POLLING_INTERVAL_DEFAULT_SECONDS
  }

  return Math.min(
    POLLING_INTERVAL_MAX_SECONDS,
    Math.max(POLLING_INTERVAL_MIN_SECONDS, Math.round(value)),
  )
}

export const usePollingStore = defineStore('polling', () => {
  const currentUserIntervalSeconds = useStorage<number>(
    CURRENT_USER_INTERVAL_STORAGE_KEY,
    POLLING_INTERVAL_DEFAULT_SECONDS,
    undefined,
    {
      // Keep the current single-tab semantics; cross-tab propagation is a separate decision.
      listenToStorageChanges: false,
      serializer: {
        // The default number read is parseFloat: an empty/corrupt value becomes NaN and an
        // out-of-range value is not clamped, so read through zod. write matches the default
        // number serializer and is only here because a custom serializer replaces both sides.
        read: (raw) =>
          parseExternalValueOrDefault(
            storedPollingIntervalSecondsSchema,
            raw,
            POLLING_INTERVAL_DEFAULT_SECONDS,
          ),
        write: (value) => String(value),
      },
    },
  )

  const currentUserIntervalMs = computed(
    () => currentUserIntervalSeconds.value * 1000,
  )

  function setCurrentUserIntervalSeconds(value: number) {
    currentUserIntervalSeconds.value = normalizePollingIntervalSeconds(value)
  }

  return {
    currentUserIntervalSeconds,
    currentUserIntervalMs,
    setCurrentUserIntervalSeconds,
  }
})
