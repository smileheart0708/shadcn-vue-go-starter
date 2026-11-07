import { useDocumentVisibility } from '@vueuse/core'
import { computed, onWatcherCleanup, ref, watch, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { resolveAPIErrorMessage } from '@/lib/api/error-messages'
import {
  openSystemLogsStream,
  type SystemLogHistoryLimit,
} from '@/lib/api/system-logs'
import {
  createSystemLogStreamPresentation,
  getReconnectDelayMs,
  historyLimitRank,
  reduceSystemLogStreamPresentation,
  shouldRetryStreamError,
  STREAM_CONNECTION_STABLE_MS,
  SystemLogBuffer,
  type SystemLogStreamEvent,
} from '@/utils/system-logs/console'

interface UseSystemLogStreamOptions {
  historyLimit: Ref<SystemLogHistoryLimit>
}

export function useSystemLogStream(options: UseSystemLogStreamOptions) {
  const { t } = useI18n()
  const documentVisibility = useDocumentVisibility()

  // `ref` (not `shallowRef`) so that the buffer's in-place mutations stay reactive.
  const buffer = ref(new SystemLogBuffer())
  const entries = computed(() => buffer.value.items)
  // One state source: the reducer owns the transitions, the three flags are projections of
  // it. Mirrored refs would need a manual write per event and desync on the next one added.
  const presentation = ref(createSystemLogStreamPresentation())
  const loading = computed(() => presentation.value.loading)
  const connecting = computed(() => presentation.value.phase === 'connecting')
  const connected = computed(() => presentation.value.phase === 'connected')
  const streamError = computed(() => presentation.value.error)
  const reconnectKey = ref(0)
  const pageVisible = computed(() => documentVisibility.value === 'visible')

  let abortController: AbortController | null = null
  let stableTimer: ReturnType<typeof setTimeout> | null = null

  function dispatch(event: SystemLogStreamEvent) {
    presentation.value = reduceSystemLogStreamPresentation(
      presentation.value,
      event,
    )
  }

  function clearStableTimer() {
    if (stableTimer === null) {
      return
    }

    clearTimeout(stableTimer)
    stableTimer = null
  }

  watch(
    () => [pageVisible.value, reconnectKey.value] as const,
    ([visible]) => {
      if (!visible) {
        disconnect()
        return
      }

      const sessionController = new AbortController()
      onWatcherCleanup(() => {
        sessionController.abort()
        disconnect()
      })
      void maintainStreamConnection(sessionController.signal)
    },
    { immediate: true },
  )

  watch(options.historyLimit, (nextLimit, previousLimit) => {
    if (historyLimitRank(nextLimit) > historyLimitRank(previousLimit)) {
      reconnect()
    }
  })

  function disconnect() {
    const controller = abortController
    abortController = null
    controller?.abort()
    clearStableTimer()
  }

  function clearEntries() {
    buffer.value.clear()
  }

  function reconnect() {
    reconnectKey.value += 1
  }

  async function maintainStreamConnection(sessionSignal: AbortSignal) {
    let retryAttempt = 0
    dispatch({ type: 'start' })

    while (!sessionSignal.aborted) {
      const controller = new AbortController()
      abortController = controller
      let failed = false
      let failure: unknown

      try {
        await openSystemLogsStream({
          tail: options.historyLimit.value,
          signal: controller.signal,
          onOpen() {
            if (!isCurrentStream(controller)) {
              return
            }

            scheduleStable(controller, () => {
              retryAttempt = 0
              dispatch({ type: 'stable' })
            })
          },
          onEntry(entry) {
            if (!isCurrentStream(controller)) {
              return
            }

            buffer.value.append(entry)
            dispatch({ type: 'entry' })
          },
        })
      } catch (error) {
        failed = true
        failure = error
      } finally {
        clearStableTimer()
        if (abortController === controller) {
          abortController = null
        }
      }

      // Normal end is as disconnected as a throw; the old branch was unreachable because
      // `finally` cleared `abortController`, so this check uses the controller's own signal.
      if (shouldStopStream(controller, sessionSignal)) {
        return
      }

      const message = failed
        ? (resolveAPIErrorMessage(t, failure) ??
          t('apiError.system_log_stream_unavailable'))
        : t('apiError.system_log_stream_unavailable')

      // `fatal` (not `failed`) so the badge and the toast carry *this* error: the reducer
      // keeps the first retry message, which by now may describe a different attempt.
      if (failed && !shouldRetryStreamError(failure)) {
        dispatch({ type: 'fatal', message })
        toast.error(message)
        return
      }

      dispatch({ type: 'failed', message })
      retryAttempt += 1
      if (
        !(await waitForReconnect(
          getReconnectDelayMs(retryAttempt),
          sessionSignal,
        ))
      ) {
        return
      }
    }
  }

  function scheduleStable(controller: AbortController, onStable: () => void) {
    clearStableTimer()
    stableTimer = setTimeout(() => {
      stableTimer = null
      if (!isCurrentStream(controller)) {
        return
      }

      onStable()
    }, STREAM_CONNECTION_STABLE_MS)
  }

  function isCurrentStream(controller: AbortController) {
    return abortController === controller && !controller.signal.aborted
  }

  // Reading the flags through a call keeps the loop's `!aborted` narrowing from leaking
  // into the post-await check.
  function shouldStopStream(
    controller: AbortController,
    sessionSignal: AbortSignal,
  ) {
    return sessionSignal.aborted || controller.signal.aborted
  }

  return {
    entries,
    loading,
    connecting,
    connected,
    streamError,
    clearEntries,
    reconnect,
  }
}

async function waitForReconnect(
  delayMs: number,
  signal: AbortSignal,
): Promise<boolean> {
  return new Promise((resolve) => {
    if (signal.aborted) {
      resolve(false)
      return
    }

    const timeoutId = window.setTimeout(() => {
      signal.removeEventListener('abort', handleAbort)
      resolve(true)
    }, delayMs)

    function handleAbort() {
      window.clearTimeout(timeoutId)
      resolve(false)
    }

    signal.addEventListener('abort', handleAbort, { once: true })
  })
}
