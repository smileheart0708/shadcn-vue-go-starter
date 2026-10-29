import { APIError } from '@/lib/api/client'
import type { BadgeVariants } from '@/components/ui/badge'
import {
  isSystemLogHistoryLimit,
  isSystemLogLevel,
  type SystemLogEntry,
  type SystemLogHistoryLimit,
  type SystemLogLevel,
} from '@/lib/api/system-logs'

const LOCAL_STREAM_MAX_BYTES = 1 * 1024 * 1024
const STREAM_ENTRY_BASE_BYTES = 64
const RECONNECT_BASE_DELAY_MS = 1_000
const RECONNECT_MAX_DELAY_MS = 30_000

interface SelectVisibleSystemLogEntriesOptions {
  entries: readonly SystemLogEntry[]
  historyLimit: SystemLogHistoryLimit
  levels: readonly SystemLogLevel[]
  searchQuery: string
}

export function selectVisibleSystemLogEntries(
  options: SelectVisibleSystemLogEntriesOptions,
): SystemLogEntry[] {
  const normalizedQuery = options.searchQuery.trim().toLowerCase()
  const selectedLevels = new Set(options.levels)

  const filteredEntries = options.entries.filter((entry) => {
    if (!isSystemLogLevel(entry.level) || !selectedLevels.has(entry.level)) {
      return false
    }

    if (normalizedQuery === '') {
      return true
    }

    return [entry.text, entry.message, entry.source].some((value) =>
      value.toLowerCase().includes(normalizedQuery),
    )
  })

  return applyHistoryLimit(filteredEntries, options.historyLimit)
}

export function applyHistoryLimit(
  sourceEntries: readonly SystemLogEntry[],
  limit: SystemLogHistoryLimit,
): SystemLogEntry[] {
  if (limit === 'ALL') {
    return [...sourceEntries]
  }

  return sourceEntries.slice(-limit)
}

/**
 * Append-only buffer for the live log stream, kept sorted by entry id and capped by an
 * approximate byte budget.
 *
 * Mutates in place and tracks the running byte total incrementally. The previous
 * implementation rebuilt the array and re-summed every entry on each append, which is
 * O(n) per SSE message — noticeable once the buffer holds tens of thousands of lines
 * under a busy stream. Vue's reactivity tracks in-place array mutation, so callers that
 * hold this in a `ref` still re-render correctly.
 */
export class SystemLogBuffer {
  private readonly entries: SystemLogEntry[] = []
  private bufferedBytes = 0

  get items(): readonly SystemLogEntry[] {
    return this.entries
  }

  append(entry: SystemLogEntry): void {
    const insertIndex = findEntryInsertIndex(this.entries, entry.id)

    // The stream can redeliver entries after a reconnect; ids are unique and ascending,
    // so a hit at the insert position means we already have this one.
    if (this.entries[insertIndex]?.id === entry.id) {
      return
    }

    this.entries.splice(insertIndex, 0, entry)
    this.bufferedBytes += getSystemLogEntrySize(entry)
    this.pruneOldest()
  }

  clear(): void {
    this.entries.length = 0
    this.bufferedBytes = 0
  }

  private pruneOldest(): void {
    while (
      this.bufferedBytes > LOCAL_STREAM_MAX_BYTES &&
      this.entries.length > 0
    ) {
      const removedEntry = this.entries.shift()
      if (!removedEntry) {
        break
      }

      this.bufferedBytes -= getSystemLogEntrySize(removedEntry)
    }
  }
}

export function normalizeHistoryLimitSelectValue(
  value: unknown,
): SystemLogHistoryLimit | null {
  if (isSystemLogHistoryLimit(value)) {
    return value
  }
  if (typeof value !== 'string') {
    return null
  }

  const normalizedValue = value.trim().toUpperCase()
  if (normalizedValue === 'ALL') {
    return 'ALL'
  }

  const numericValue = Number(normalizedValue)
  return isSystemLogHistoryLimit(numericValue) ? numericValue : null
}

export function historyLimitRank(limit: SystemLogHistoryLimit): number {
  return limit === 'ALL' ? Number.POSITIVE_INFINITY : limit
}

export function getLevelBadgeVariant(
  level: string,
): NonNullable<BadgeVariants['variant']> {
  switch (level) {
    case 'ERROR':
      return 'destructive'
    case 'WARN':
      return 'warning'
    case 'INFO':
      return 'outline'
    default:
      return 'secondary'
  }
}

export function getConnectionIndicatorClass(
  connecting: boolean,
  connected: boolean,
): string {
  if (connecting) {
    return 'bg-amber-500'
  }
  if (connected) {
    return 'bg-emerald-500'
  }
  return 'bg-red-500'
}

export function shouldRetryStreamError(error: unknown): boolean {
  if (error instanceof APIError) {
    return error.status === 408 || error.status === 429 || error.status >= 500
  }

  return error instanceof TypeError
}

export function getReconnectDelayMs(attempt: number): number {
  const exponent = Math.max(0, attempt - 1)
  return Math.min(
    RECONNECT_MAX_DELAY_MS,
    RECONNECT_BASE_DELAY_MS * 2 ** exponent,
  )
}

export function formatSystemLogTimestamp(
  timestamp: number | null | undefined,
): string {
  if (timestamp === null || timestamp === undefined) {
    return ''
  }

  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(new Date(timestamp * 1000))
}

function findEntryInsertIndex(
  sourceEntries: readonly SystemLogEntry[],
  entryId: number,
): number {
  let low = 0
  let high = sourceEntries.length
  while (low < high) {
    const middle = Math.floor((low + high) / 2)
    const middleEntry = sourceEntries[middle]
    if (middleEntry.id < entryId) {
      low = middle + 1
    } else {
      high = middle
    }
  }
  return low
}

function getSystemLogEntrySize(entry: SystemLogEntry): number {
  return (
    STREAM_ENTRY_BASE_BYTES +
    entry.level.length +
    entry.message.length +
    entry.text.length +
    entry.source.length
  )
}
