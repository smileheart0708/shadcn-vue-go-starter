import { createParser, type EventSourceMessage } from 'eventsource-parser'
import { z } from 'zod'
import { APIError, authApi, normalizeAPIError } from '@/lib/api/client'

export const SYSTEM_LOG_LEVEL_VALUES = [
  'DEBUG',
  'INFO',
  'WARN',
  'ERROR',
] as const
export const SYSTEM_LOG_HISTORY_LIMIT_VALUES = [
  100,
  200,
  500,
  1000,
  'ALL',
] as const

const systemLogLevelSet = new Set<string>(SYSTEM_LOG_LEVEL_VALUES)
const systemLogHistoryLimitSet = new Set<unknown>(
  SYSTEM_LOG_HISTORY_LIMIT_VALUES,
)

export type SystemLogLevel = (typeof SYSTEM_LOG_LEVEL_VALUES)[number]
export type SystemLogHistoryLimit =
  (typeof SYSTEM_LOG_HISTORY_LIMIT_VALUES)[number]

export const systemLogEntrySchema = z.object({
  id: z.number().int().positive(),
  timestamp: z.number().int().nonnegative(),
  level: z.string().min(1),
  message: z.string(),
  text: z.string(),
  source: z.string().min(1),
})

export type SystemLogEntry = z.infer<typeof systemLogEntrySchema>

interface OpenSystemLogsStreamOptions {
  tail?: SystemLogHistoryLimit
  signal: AbortSignal
  onOpen?: () => void
  onEntry: (entry: SystemLogEntry) => void
}

export async function openSystemLogsStream(
  options: OpenSystemLogsStreamOptions,
): Promise<void> {
  try {
    const response = await authApi.get('/api/management/system-logs/stream', {
      headers: {
        Accept: 'text/event-stream',
      },
      searchParams: {
        tail: formatSystemLogHistoryLimit(options.tail ?? 'ALL'),
      },
      signal: options.signal,
    })

    if (!response.body) {
      throw new APIError(
        500,
        'System log stream is unavailable.',
        'system_log_stream_unavailable',
      )
    }

    options.onOpen?.()

    const parser = createParser({
      onEvent(event) {
        handleStreamEvent(event, options.onEntry)
      },
    })

    const reader = response.body.getReader()
    const decoder = new TextDecoder()

    for (;;) {
      const { done, value } = await reader.read()
      if (done) {
        break
      }

      parser.feed(decoder.decode(value, { stream: true }))
    }

    parser.feed(decoder.decode())
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export function isSystemLogLevel(value: string): value is SystemLogLevel {
  return systemLogLevelSet.has(value)
}

export function isSystemLogHistoryLimit(
  value: unknown,
): value is SystemLogHistoryLimit {
  return systemLogHistoryLimitSet.has(value)
}

export function formatSystemLogHistoryLimit(
  value: SystemLogHistoryLimit,
): string {
  return value === 'ALL' ? 'all' : String(value)
}

/**
 * Parses one SSE frame, returning `null` when this build cannot read it.
 *
 * A throw here would escape `parser.feed()` into `openSystemLogsStream`'s catch, where a
 * malformed frame is indistinguishable from a dropped connection: `shouldRetryStreamError`
 * only recognises `APIError` / `TypeError`, so one bad line (a truncated write, a proxy
 * injecting its own body, a contract change) would end the whole session instead of
 * dropping that line. Losing a frame is recoverable; losing the stream is not.
 */
export function parseSystemLogEntry(data: string): SystemLogEntry | null {
  let payload: unknown
  try {
    payload = JSON.parse(data)
  } catch {
    return null
  }

  const parsed = systemLogEntrySchema.safeParse(payload)
  return parsed.success ? parsed.data : null
}

function handleStreamEvent(
  event: EventSourceMessage,
  onEntry: (entry: SystemLogEntry) => void,
) {
  if (event.event !== 'log') {
    return
  }

  const entry = parseSystemLogEntry(event.data)
  if (entry !== null) {
    onEntry(entry)
  }
}
