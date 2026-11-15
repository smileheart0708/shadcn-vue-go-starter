import type { SystemLogEntry } from '@/lib/api/system-logs'
import type { SystemLogExportFormat } from '@/stores/system-logs-preferences'
import { formatSystemLogTimestamp } from './console'

/**
 * Serializes entries the caller already filtered.
 *
 * Do not add level/search/history filtering here: the visible list and the export
 * must come from the same `selectVisibleSystemLogEntries` result, otherwise the
 * dialog counts and downloads a set the user is not looking at. Keeping the filter
 * in one place is the only thing that keeps those two in sync.
 */
export function createSystemLogExportBlob(
  entries: readonly SystemLogEntry[],
  format: SystemLogExportFormat,
): Blob {
  switch (format) {
    case 'json':
      return new Blob([JSON.stringify(entries, null, 2)], {
        type: 'application/json;charset=utf-8',
      })
    case 'txt':
      return new Blob([serializeSystemLogsTXT(entries)], {
        type: 'text/plain;charset=utf-8',
      })
    case 'csv':
    default:
      return new Blob([`\uFEFF${serializeSystemLogsCSV(entries)}`], {
        type: 'text/csv;charset=utf-8',
      })
  }
}

export function getSystemLogExportFileName(
  format: SystemLogExportFormat,
): string {
  return `system-logs-${formatExportFileDate(new Date())}.${format}`
}

function serializeSystemLogsCSV(entries: readonly SystemLogEntry[]): string {
  const header = ['timestamp', 'level', 'source', 'message', 'text']
  const rows = entries.map((entry) => [
    formatSystemLogTimestamp(entry.timestamp),
    entry.level,
    entry.source,
    entry.message,
    entry.text,
  ])

  return [header, ...rows]
    .map((row) => row.map(escapeCSVValue).join(','))
    .join('\r\n')
}

function serializeSystemLogsTXT(entries: readonly SystemLogEntry[]): string {
  return entries
    .map(
      (entry) =>
        `[${formatSystemLogTimestamp(entry.timestamp)}] [${entry.level}] [${entry.source}] ${entry.text}`,
    )
    .join('\r\n')
}

function escapeCSVValue(value: string): string {
  return `"${value.replace(/"/g, '""')}"`
}

function formatExportFileDate(date: Date): string {
  const year = String(date.getFullYear())
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')

  return `${year}${month}${day}-${hours}${minutes}${seconds}`
}
