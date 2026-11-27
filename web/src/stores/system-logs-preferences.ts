import { useStorage } from '@vueuse/core'
import { computed } from 'vue'
import { defineStore } from 'pinia'
import { z } from 'zod'
import { parseExternalJsonOrDefault } from '@/lib/external-input'
import {
  isSystemLogLevel,
  SYSTEM_LOG_LEVEL_VALUES,
  type SystemLogHistoryLimit,
  type SystemLogLevel,
} from '@/lib/api/system-logs'

const STORAGE_KEY = 'app.system-logs.preferences'

export const SYSTEM_LOG_EXPORT_FORMAT_VALUES = ['csv', 'txt', 'json'] as const

export type SystemLogExportFormat =
  (typeof SYSTEM_LOG_EXPORT_FORMAT_VALUES)[number]

const systemLogExportFormatSet = new Set<string>(
  SYSTEM_LOG_EXPORT_FORMAT_VALUES,
)

interface SystemLogsPreferences {
  levels: SystemLogLevel[]
  historyLimit: SystemLogHistoryLimit
  exportFormat: SystemLogExportFormat
}

const defaultPreferences: SystemLogsPreferences = {
  levels: [...SYSTEM_LOG_LEVEL_VALUES],
  historyLimit: 'ALL',
  exportFormat: 'csv',
}

const storedPreferencesSchema = z.object({
  levels: z.array(z.string()).optional(),
  historyLimit: z
    .union([
      z.literal(100),
      z.literal(200),
      z.literal(500),
      z.literal(1000),
      z.literal('ALL'),
    ])
    .optional(),
  exportFormat: z.string().optional(),
})

export const useSystemLogsPreferencesStore = defineStore(
  'system-logs-preferences',
  () => {
    const preferences = useStorage<SystemLogsPreferences>(
      STORAGE_KEY,
      defaultPreferences,
      undefined,
      {
        // Keep the current single-tab semantics; cross-tab propagation is a separate decision.
        listenToStorageChanges: false,
        serializer: {
          // The default object read throws on corrupt JSON and skips the level/format
          // normalization, which would surface a partial payload as-is.
          read: readStoredPreferences,
          write: (value) => JSON.stringify(value),
        },
      },
    )

    const levels = computed(() => preferences.value.levels)
    const historyLimit = computed(() => preferences.value.historyLimit)
    const exportFormat = computed(() => preferences.value.exportFormat)

    function setLevels(nextLevels: readonly SystemLogLevel[]) {
      preferences.value = {
        ...preferences.value,
        levels: normalizeLevels(nextLevels),
      }
    }

    function setHistoryLimit(nextHistoryLimit: SystemLogHistoryLimit) {
      preferences.value = {
        ...preferences.value,
        historyLimit: nextHistoryLimit,
      }
    }

    function setExportFormat(nextExportFormat: SystemLogExportFormat) {
      preferences.value = {
        ...preferences.value,
        exportFormat: nextExportFormat,
      }
    }

    return {
      levels,
      historyLimit,
      exportFormat,
      setLevels,
      setHistoryLimit,
      setExportFormat,
    }
  },
)

function readStoredPreferences(rawValue: string): SystemLogsPreferences {
  const parsedPayload = parseExternalJsonOrDefault(
    storedPreferencesSchema,
    rawValue,
    {},
  )
  return {
    levels: normalizeLevels(parsedPayload.levels ?? defaultPreferences.levels),
    historyLimit: parsedPayload.historyLimit ?? defaultPreferences.historyLimit,
    exportFormat: normalizeExportFormat(parsedPayload.exportFormat),
  }
}

function normalizeLevels(values: readonly string[]): SystemLogLevel[] {
  const selectedLevels = new Set(values.filter(isSystemLogLevel))
  return [...SYSTEM_LOG_LEVEL_VALUES].filter((level) =>
    selectedLevels.has(level),
  )
}

function normalizeExportFormat(
  value: string | undefined,
): SystemLogExportFormat {
  return isSystemLogExportFormat(value)
    ? value
    : defaultPreferences.exportFormat
}

function isSystemLogExportFormat(
  value: string | undefined,
): value is SystemLogExportFormat {
  return value !== undefined && systemLogExportFormatSet.has(value)
}
