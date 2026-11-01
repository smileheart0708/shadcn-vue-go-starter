import { http, HttpResponse } from 'msw'
import { CAPABILITY } from '@/lib/auth/capabilities.generated'
import { hasCapability } from '@/mocks/handlers/permissions'
import { requireAuthenticated } from '@/mocks/handlers/session'
import { jsonError, state } from '@/mocks/handlers/state'

type SystemLogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'

function createSystemLogEntry(
  id: number,
  level: SystemLogLevel,
  source: string,
  text: string,
) {
  return {
    id,
    timestamp: Math.floor(Date.now() / 1000),
    level,
    source,
    message: text,
    text,
  }
}

const baseSystemLogs = [
  createSystemLogEntry(1, 'INFO', 'mock', 'Mock system log stream connected.'),
  createSystemLogEntry(2, 'INFO', 'auth', 'Mock owner session initialized.'),
  createSystemLogEntry(3, 'DEBUG', 'setup', 'Mock install state is completed.'),
]

function formatSystemLogEvent(entry: ReturnType<typeof createSystemLogEntry>) {
  return `event: log\nid: ${String(entry.id)}\ndata: ${JSON.stringify(entry)}\n\n`
}

function selectSystemLogReplayEntries(rawTail: string | null) {
  const normalizedTail = rawTail?.trim().toLowerCase()
  if (
    normalizedTail === undefined ||
    normalizedTail === '' ||
    normalizedTail === 'all'
  ) {
    return baseSystemLogs
  }

  const tail = Number(normalizedTail)
  if (tail === 100 || tail === 200 || tail === 500 || tail === 1000) {
    return baseSystemLogs.slice(-tail)
  }

  return baseSystemLogs
}

function buildSystemLogStreamResponse(rawTail: string | null) {
  const encoder = new TextEncoder()
  let intervalId: ReturnType<typeof setInterval> | null = null
  const replayEntries = selectSystemLogReplayEntries(rawTail)

  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      for (const entry of replayEntries) {
        controller.enqueue(encoder.encode(formatSystemLogEvent(entry)))
      }

      intervalId = setInterval(() => {
        const entry = createSystemLogEntry(
          state.nextLogId++,
          'INFO',
          'mock',
          'Mock system log placeholder heartbeat.',
        )
        controller.enqueue(encoder.encode(formatSystemLogEvent(entry)))
      }, 5_000)
    },
    cancel() {
      if (intervalId !== null) {
        clearInterval(intervalId)
      }
    },
  })

  return new HttpResponse(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      Connection: 'keep-alive',
    },
  })
}

export const systemLogsHandlers = [
  http.get('/api/management/system-logs/stream', ({ request }) => {
    const { error, user } = requireAuthenticated(request)
    if (error) {
      return error
    }
    if (!hasCapability(user, CAPABILITY.managementSystemLogsRead)) {
      return jsonError(403, 'forbidden', 'Forbidden.')
    }

    const url = new URL(request.url)
    return buildSystemLogStreamResponse(url.searchParams.get('tail'))
  }),
]
