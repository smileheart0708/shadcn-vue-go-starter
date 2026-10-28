import { readonly, ref, toValue, type MaybeRefOrGetter, type Ref } from 'vue'

export type PollingRunReason = 'interval' | 'manual' | 'resume'

interface PollingRunContext {
  reason: PollingRunReason
  signal: AbortSignal
}

type MaybePromise<T> = T | Promise<T>

export interface PollingTaskOptions<TData> {
  key: string
  intervalMs: MaybeRefOrGetter<number>
  maxBackoffMs?: number
  enabled?: () => boolean
  fetch: (context: PollingRunContext) => Promise<TData>
  apply: (data: TData, context: PollingRunContext) => MaybePromise<void>
}

export interface PollingTaskState {
  refreshing: Readonly<Ref<boolean>>
  paused: Readonly<Ref<boolean>>
  running: Readonly<Ref<boolean>>
  lastUpdatedAt: Readonly<Ref<number | null>>
  error: Readonly<Ref<unknown>>
}

export interface PollingTaskHandle extends PollingTaskState {
  start: () => void
  stop: () => void
  pause: () => void
  resume: () => void
  refreshNow: () => void
}

interface PollingTaskSubscription extends PollingTaskHandle {
  release: () => void
}

interface InternalPollingTaskOptions {
  key: string
  intervalMs: MaybeRefOrGetter<number>
  maxBackoffMs?: number
  enabled?: () => boolean
  execute: (context: PollingRunContext) => Promise<void>
}

const DEFAULT_MAX_BACKOFF_MS = 60_000

/**
 * Tasks are keyed and ref-counted so several components can subscribe to the same poll
 * without multiplying requests: the first consumer creates the timer, the last one to
 * release it tears the timer down. Tasks also pause while the document is hidden and
 * back off exponentially on failure.
 */
const taskRegistry = new Map<string, InternalPollingTask>()

let visibilityListenerInstalled = false

class InternalPollingTask {
  private readonly key: string
  private options: InternalPollingTaskOptions
  private consumerCount = 0
  private timer: ReturnType<typeof setTimeout> | null = null
  private abortController: AbortController | null = null
  private runId = 0
  private inFlight = false
  private failureCount = 0
  private manualPaused = false
  private pendingImmediateReason: PollingRunReason | null = null
  private disposed = false

  readonly refreshing = ref(false)
  readonly paused = ref(false)
  readonly running = ref(false)
  readonly lastUpdatedAt = ref<number | null>(null)
  readonly error = ref<unknown>(null)

  constructor(options: InternalPollingTaskOptions) {
    this.key = options.key
    this.options = options
  }

  acquire(options: InternalPollingTaskOptions): PollingTaskSubscription {
    this.options = options
    this.consumerCount += 1

    let released = false

    return {
      refreshing: readonly(this.refreshing),
      paused: readonly(this.paused),
      running: readonly(this.running),
      lastUpdatedAt: readonly(this.lastUpdatedAt),
      error: readonly(this.error),
      start: () => {
        this.start()
      },
      stop: () => {
        this.stop()
      },
      pause: () => {
        this.pause()
      },
      resume: () => {
        this.resume()
      },
      refreshNow: () => {
        this.refreshNow()
      },
      release: () => {
        if (released) {
          return
        }

        released = true
        this.release()
      },
    }
  }

  handleVisibilityChange() {
    if (this.disposed) {
      return
    }

    this.syncPausedState()

    if (!this.running.value || this.manualPaused) {
      return
    }

    if (!isDocumentVisible()) {
      this.clearTimer()
      this.abortInFlight()
      this.refreshing.value = false
      return
    }

    this.pendingImmediateReason ??= 'resume'

    this.flushImmediateRun()
  }

  private release() {
    this.consumerCount = Math.max(0, this.consumerCount - 1)

    if (this.consumerCount > 0) {
      return
    }

    this.dispose()
    taskRegistry.delete(this.key)
    maybeRemoveVisibilityListener()
  }

  private dispose() {
    if (this.disposed) {
      return
    }

    this.disposed = true
    this.stop()
  }

  private start() {
    if (this.disposed || this.running.value) {
      return
    }

    this.running.value = true
    this.manualPaused = false

    // The first run waits one interval instead of firing immediately: tasks are started
    // from components that already hold freshly loaded data (the app shell mounts right
    // after the router guard fetched the viewer), so an immediate run repeats that
    // request. `refreshNow()`/`resume()` remain the explicit ways to refresh on demand.
    this.syncPausedState()
    this.scheduleNext(this.getSuccessDelayMs(this.options))
  }

  private stop() {
    if (this.disposed && !this.running.value && !this.inFlight) {
      return
    }

    this.running.value = false
    this.manualPaused = false
    this.pendingImmediateReason = null
    this.clearTimer()
    this.abortInFlight()
    this.refreshing.value = false
    this.paused.value = false
  }

  private pause() {
    if (this.disposed || !this.running.value) {
      return
    }

    this.manualPaused = true
    this.pendingImmediateReason = null
    this.clearTimer()
    this.abortInFlight()
    this.refreshing.value = false
    this.syncPausedState()
  }

  private resume() {
    if (this.disposed || !this.running.value) {
      return
    }

    this.manualPaused = false
    this.requestImmediateRun('resume')
  }

  private refreshNow() {
    if (this.disposed || !this.running.value) {
      return
    }

    this.requestImmediateRun('manual')
  }

  private requestImmediateRun(reason: PollingRunReason) {
    this.pendingImmediateReason = reason
    this.clearTimer()
    this.flushImmediateRun()
  }

  private flushImmediateRun() {
    if (
      this.pendingImmediateReason === null ||
      this.inFlight ||
      !this.running.value
    ) {
      this.syncPausedState()
      return
    }

    this.syncPausedState()

    if (this.paused.value) {
      return
    }

    const reason = this.pendingImmediateReason
    this.pendingImmediateReason = null
    void this.execute(reason)
  }

  private async execute(reason: PollingRunReason) {
    if (this.disposed || this.inFlight || !this.running.value) {
      return
    }

    this.syncPausedState()

    if (this.paused.value) {
      return
    }

    const options = this.options
    const controller = new AbortController()
    const runId = ++this.runId

    this.abortController = controller
    this.inFlight = true
    this.refreshing.value = true

    let nextDelayMs: number | null = null

    try {
      await options.execute({ reason, signal: controller.signal })

      if (
        !controller.signal.aborted &&
        runId === this.runId &&
        this.isRunning()
      ) {
        this.failureCount = 0
        this.lastUpdatedAt.value = Date.now()
        this.error.value = null
        nextDelayMs = this.getSuccessDelayMs(options)
      }
    } catch (error) {
      if (
        !isAbortError(error) &&
        !controller.signal.aborted &&
        runId === this.runId &&
        this.isRunning()
      ) {
        this.failureCount += 1
        this.error.value = error
        nextDelayMs = this.getFailureDelayMs(options)
      }
    } finally {
      this.inFlight = false
      this.refreshing.value = false

      if (this.abortController === controller) {
        this.abortController = null
      }

      this.syncPausedState()
    }

    if (this.hasStopped()) {
      return
    }

    if (this.pendingImmediateReason !== null) {
      this.flushImmediateRun()
      return
    }

    if (nextDelayMs === null || this.isPaused()) {
      return
    }

    this.scheduleNext(nextDelayMs)
  }

  private hasStopped() {
    return !this.isRunning() || this.disposed
  }

  private isPaused() {
    return this.paused.value
  }

  private isRunning() {
    return this.running.value
  }

  private scheduleNext(delayMs: number) {
    this.clearTimer()

    if (this.disposed || !this.running.value) {
      return
    }

    this.timer = setTimeout(
      () => {
        this.timer = null
        this.requestImmediateRun('interval')
      },
      Math.max(0, delayMs),
    )
  }

  private abortInFlight() {
    const controller = this.abortController
    if (!controller) {
      return
    }

    this.abortController = null
    controller.abort()
  }

  private clearTimer() {
    if (this.timer === null) {
      return
    }

    clearTimeout(this.timer)
    this.timer = null
  }

  private syncPausedState() {
    this.paused.value =
      this.running.value &&
      (this.manualPaused || !isDocumentVisible() || !this.isEnabled())
  }

  private isEnabled() {
    return this.options.enabled?.() ?? true
  }

  private getSuccessDelayMs(options: InternalPollingTaskOptions) {
    return Math.max(0, toValue(options.intervalMs))
  }

  private getFailureDelayMs(options: InternalPollingTaskOptions) {
    const baseDelayMs = Math.max(0, toValue(options.intervalMs))
    const exponent = Math.max(0, this.failureCount - 1)
    const nextDelayMs = baseDelayMs * 2 ** exponent
    return Math.min(options.maxBackoffMs ?? DEFAULT_MAX_BACKOFF_MS, nextDelayMs)
  }
}

export function acquirePollingTask<TData>(
  options: PollingTaskOptions<TData>,
): PollingTaskSubscription {
  const normalizedOptions = toInternalPollingTaskOptions(options)
  let task = taskRegistry.get(options.key)

  if (!task) {
    task = new InternalPollingTask(normalizedOptions)
    taskRegistry.set(options.key, task)
    ensureVisibilityListener()
  }

  return task.acquire(normalizedOptions)
}

function ensureVisibilityListener() {
  if (visibilityListenerInstalled || typeof document === 'undefined') {
    return
  }

  document.addEventListener('visibilitychange', handleVisibilityChange)
  visibilityListenerInstalled = true
}

function maybeRemoveVisibilityListener() {
  if (
    !visibilityListenerInstalled ||
    taskRegistry.size > 0 ||
    typeof document === 'undefined'
  ) {
    return
  }

  document.removeEventListener('visibilitychange', handleVisibilityChange)
  visibilityListenerInstalled = false
}

function handleVisibilityChange() {
  for (const task of taskRegistry.values()) {
    task.handleVisibilityChange()
  }
}

function isDocumentVisible() {
  return (
    typeof document === 'undefined' || document.visibilityState === 'visible'
  )
}

function isAbortError(error: unknown) {
  return error instanceof DOMException
    ? error.name === 'AbortError'
    : error instanceof Error && error.name === 'AbortError'
}

function toInternalPollingTaskOptions<TData>(
  options: PollingTaskOptions<TData>,
): InternalPollingTaskOptions {
  return {
    key: options.key,
    intervalMs: options.intervalMs,
    maxBackoffMs: options.maxBackoffMs,
    enabled: options.enabled,
    execute: async (context) => {
      const data = await options.fetch(context)
      await options.apply(data, context)
    },
  }
}
