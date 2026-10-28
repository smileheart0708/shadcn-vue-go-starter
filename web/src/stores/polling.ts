import { computed, ref } from 'vue'
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

function readStoredCurrentUserIntervalSeconds() {
  if (typeof window === 'undefined') {
    return POLLING_INTERVAL_DEFAULT_SECONDS
  }

  const rawValue = window.localStorage.getItem(
    CURRENT_USER_INTERVAL_STORAGE_KEY,
  )
  return parseExternalValueOrDefault(
    storedPollingIntervalSecondsSchema,
    rawValue,
    POLLING_INTERVAL_DEFAULT_SECONDS,
  )
}

function persistCurrentUserIntervalSeconds(value: number) {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.setItem(CURRENT_USER_INTERVAL_STORAGE_KEY, String(value))
}

export const usePollingStore = defineStore('polling', () => {
  const initialCurrentUserIntervalSeconds =
    readStoredCurrentUserIntervalSeconds()
  const currentUserIntervalSeconds = ref(initialCurrentUserIntervalSeconds)

  persistCurrentUserIntervalSeconds(initialCurrentUserIntervalSeconds)

  const currentUserIntervalMs = computed(
    () => currentUserIntervalSeconds.value * 1000,
  )

  function setCurrentUserIntervalSeconds(value: number) {
    const nextValue = normalizePollingIntervalSeconds(value)

    if (currentUserIntervalSeconds.value === nextValue) {
      persistCurrentUserIntervalSeconds(nextValue)
      return
    }

    currentUserIntervalSeconds.value = nextValue
    persistCurrentUserIntervalSeconds(nextValue)
  }

  return {
    currentUserIntervalSeconds,
    currentUserIntervalMs,
    setCurrentUserIntervalSeconds,
  }
})
