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
  getReconnectDelayMs,
  historyLimitRank,
  shouldRetryStreamError,
  SystemLogBuffer,
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
  const loading = ref(true)
  const connecting = ref(false)
  const connected = ref(false)
  const streamError = ref('')
  const reconnectKey = ref(0)
  const pageVisible = computed(() => documentVisibility.value === 'visible')

  let abortController: AbortController | null = null

  watch(
    () => [pageVisible.value, reconnectKey.value] as const,
    ([visible]) => {
      if (!visible) {
        disconnect()
        loading.value = entries.value.length === 0
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
    connecting.value = false
    connected.value = false
  }

  function clearEntries() {
    buffer.value.clear()
  }

  function reconnect() {
    streamError.value = ''
    reconnectKey.value += 1
  }

  async function maintainStreamConnection(sessionSignal: AbortSignal) {
    let retryAttempt = 0

    while (!sessionSignal.aborted) {
      const controller = new AbortController()
      abortController = controller
      connecting.value = true
      connected.value = false
      loading.value = entries.value.length === 0

      try {
        await openSystemLogsStream({
          tail: options.historyLimit.value,
          signal: controller.signal,
          onOpen() {
            if (!isCurrentStream(controller)) {
              return
            }

            retryAttempt = 0
            connecting.value = false
            connected.value = true
            loading.value = false
            streamError.value = ''
          },
          onEntry(entry) {
            if (!isCurrentStream(controller)) {
              return
            }

            buffer.value.append(entry)
          },
        })
      } catch (error) {
        if (shouldStopStream(controller, sessionSignal)) {
          return
        }

        connecting.value = false
        connected.value = false
        loading.value = false

        const message =
          resolveAPIErrorMessage(t, error) ??
          t('apiError.system_log_stream_unavailable')
        streamError.value = message

        if (!shouldRetryStreamError(error)) {
          toast.error(message)
          return
        }

        retryAttempt += 1
        if (
          !(await waitForReconnect(
            getReconnectDelayMs(retryAttempt),
            sessionSignal,
          ))
        ) {
          return
        }
        continue
      } finally {
        if (abortController === controller) {
          abortController = null
        }
      }

      if (shouldStopStream(controller, sessionSignal)) {
        return
      }

      connecting.value = false
      connected.value = false
      loading.value = false

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

  function isCurrentStream(controller: AbortController) {
    return abortController === controller && !controller.signal.aborted
  }

  function shouldStopStream(
    controller: AbortController,
    sessionSignal: AbortSignal,
  ) {
    return sessionSignal.aborted || !isCurrentStream(controller)
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
