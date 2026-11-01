import { watch, ref } from 'vue'
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
    const initialPreferences = readStoredPreferences()
    const levels = ref<SystemLogLevel[]>(initialPreferences.levels)
    const historyLimit = ref<SystemLogHistoryLimit>(
      initialPreferences.historyLimit,
    )
    const exportFormat = ref<SystemLogExportFormat>(
      initialPreferences.exportFormat,
    )

    watch([levels, historyLimit, exportFormat], persistCurrentPreferences, {
      deep: true,
    })
    persistCurrentPreferences()

    function setLevels(nextLevels: readonly SystemLogLevel[]) {
      levels.value = normalizeLevels(nextLevels)
    }

    function setHistoryLimit(nextHistoryLimit: SystemLogHistoryLimit) {
      historyLimit.value = nextHistoryLimit
    }

    function setExportFormat(nextExportFormat: SystemLogExportFormat) {
      exportFormat.value = nextExportFormat
    }

    function persistCurrentPreferences() {
      persistPreferences({
        levels: levels.value,
        historyLimit: historyLimit.value,
        exportFormat: exportFormat.value,
      })
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

function readStoredPreferences(): SystemLogsPreferences {
  if (typeof window === 'undefined') {
    return defaultPreferences
  }

  const rawValue = window.localStorage.getItem(STORAGE_KEY)
  if (rawValue === null) {
    return defaultPreferences
  }

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

function persistPreferences(preferences: SystemLogsPreferences) {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences))
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
