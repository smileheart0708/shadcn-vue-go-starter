import { useDocumentVisibility } from '@vueuse/core'
import {
  effectScope,
  readonly,
  ref,
  toValue,
  watch,
  type EffectScope,
  type MaybeRefOrGetter,
  type Ref,
} from 'vue'

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
  /**
   * Failed runs since the last success, reset by a successful run. Consumers that surface a
   * problem should react to `=== 1` only: a failing task keeps retrying with backoff, so
   * reacting to every attempt would stack one notification per retry.
   */
  consecutiveFailures: Readonly<Ref<number>>
}

export interface PollingTaskHandle extends PollingTaskState {
  start: () => void
  stop: () => void
  pause: () => void
  resume: () => void
  refreshNow: () => void
}

/** Handle returned by `acquirePollingTask`: the public handle plus subscription teardown. */
export interface PollingTaskSubscription extends PollingTaskHandle {
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

/**
 * Whether the tab is visible, so a task can decide synchronously instead of racing an event.
 *
 * The value and its `visibilitychange` listener both come from vueuse (the system log stream
 * reads visibility the same way) — a hand-rolled listener plus a copied ref would be two
 * sources of truth. `ensureVisibilityTracking` rebinds this reader whenever tracking starts,
 * and it is only read while a task exists, i.e. always after that rebinding.
 */
let isDocumentVisible: () => boolean = () => true

let visibilityScope: EffectScope | null = null

class InternalPollingTask {
  private readonly key: string
  private options: InternalPollingTaskOptions
  private consumerCount = 0
  private timer: ReturnType<typeof setTimeout> | null = null
  private abortController: AbortController | null = null
  private runId = 0
  private inFlight = false
  private manualPaused = false
  private pendingImmediateReason: PollingRunReason | null = null
  private disposed = false
  private readonly intervalWatchScope = effectScope(true)

  readonly refreshing = ref(false)
  readonly paused = ref(false)
  readonly running = ref(false)
  readonly failureCount = ref(0)

  constructor(options: InternalPollingTaskOptions) {
    this.key = options.key
    this.options = options

    // Re-arming on an interval change is this module's own contract: consumers hand over a
    // reactive `intervalMs`, so forcing every call site to `pause()`/`resume()` around a
    // settings change would leak an implementation detail they can silently get wrong.
    this.intervalWatchScope.run(() => {
      watch(
        () => toValue(this.options.intervalMs),
        () => {
          this.handleIntervalChange()
        },
      )
    })
  }

  acquire(options: InternalPollingTaskOptions): PollingTaskSubscription {
    // The newest subscriber's options win for the whole task — see `acquirePollingTask`.
    this.options = options
    this.consumerCount += 1

    let released = false

    return {
      refreshing: readonly(this.refreshing),
      paused: readonly(this.paused),
      running: readonly(this.running),
      consecutiveFailures: readonly(this.failureCount),
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

    // A hidden tab is not a reason to serve stale data on return: flush now instead of
    // waiting out the tick that was cancelled while hidden.
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
    maybeStopVisibilityTracking()
  }

  private dispose() {
    if (this.disposed) {
      return
    }

    this.disposed = true
    this.intervalWatchScope.stop()
    this.stop()
  }

  private start() {
    if (this.disposed || this.running.value) {
      return
    }

    this.running.value = true
    this.manualPaused = false

    this.syncPausedState()

    // A `refreshNow()` issued before `start()` must not be dropped: consumers register
    // their own `onMounted` hooks, so "refresh on mount" may legitimately run first.
    if (this.pendingImmediateReason !== null) {
      this.flushImmediateRun()
      return
    }

    // The first run waits one interval instead of firing immediately: tasks are started
    // from components that already hold freshly loaded data (the app shell mounts right
    // after the router guard fetched the viewer), so an immediate run repeats that
    // request. `refreshNow()`/`resume()` remain the explicit ways to refresh on demand.
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
    if (this.disposed) {
      return
    }

    // `flushImmediateRun` keeps the pending reason while the task is not running (or is
    // paused), so ordering between this call and `start()` does not matter.
    this.requestImmediateRun('manual')
  }

  private handleIntervalChange() {
    if (this.disposed || !this.running.value) {
      return
    }

    // Nothing is armed while paused (resume and visibility changes flush immediately), and
    // an in-flight run re-computes its delay when it settles — re-arming here would
    // double-schedule the next tick.
    if (this.inFlight || this.paused.value) {
      return
    }

    this.scheduleNext(this.getNextDelayMs())
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
        this.failureCount.value = 0
        nextDelayMs = this.getSuccessDelayMs(options)
      }
    } catch (error) {
      if (
        !isAbortError(error) &&
        !controller.signal.aborted &&
        runId === this.runId &&
        this.isRunning()
      ) {
        this.failureCount.value += 1
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

  /**
   * Delay to re-arm with after an interval change: the steady-state cadence once a run has
   * succeeded, otherwise the backoff step the timer being replaced was already using.
   */
  private getNextDelayMs() {
    return this.failureCount.value > 0
      ? this.getFailureDelayMs(this.options)
      : this.getSuccessDelayMs(this.options)
  }

  private getSuccessDelayMs(options: InternalPollingTaskOptions) {
    return Math.max(0, toValue(options.intervalMs))
  }

  private getFailureDelayMs(options: InternalPollingTaskOptions) {
    const baseDelayMs = Math.max(0, toValue(options.intervalMs))
    const exponent = Math.max(0, this.failureCount.value - 1)
    const nextDelayMs = baseDelayMs * 2 ** exponent
    return Math.min(options.maxBackoffMs ?? DEFAULT_MAX_BACKOFF_MS, nextDelayMs)
  }
}

/**
 * Subscribes to the task registered under `options.key`, creating it on the first call.
 *
 * One key means one timer and one `fetch`: a later subscriber replaces the task's options
 * for every consumer, so callers sharing a key must come from the same composable. There is
 * no way to compare the closures and detect a mismatch, which is why the key is owned by
 * that composable instead of being passed in from call sites.
 */
export function acquirePollingTask<TData>(
  options: PollingTaskOptions<TData>,
): PollingTaskSubscription {
  const normalizedOptions = toInternalPollingTaskOptions(options)
  let task = taskRegistry.get(options.key)

  if (!task) {
    task = new InternalPollingTask(normalizedOptions)
    taskRegistry.set(options.key, task)
    ensureVisibilityTracking()
  }

  return task.acquire(normalizedOptions)
}

/**
 * The listener lives in a detached scope so it survives individual consumers: vueuse
 * registers its cleanup with `onScopeDispose`, so stopping the scope once the last task is
 * released removes the listener without any manual bookkeeping here.
 */
function ensureVisibilityTracking() {
  if (visibilityScope !== null) {
    return
  }

  const scope = effectScope(true)

  scope.run(() => {
    const visibility = useDocumentVisibility()

    // Rebind the reader instead of copying the value: vueuse's ref already tracks the live
    // state, and a copied value would need its own listener to stay in sync.
    isDocumentVisible = () => visibility.value === 'visible'

    watch(visibility, () => {
      for (const task of taskRegistry.values()) {
        task.handleVisibilityChange()
      }
    })
  })

  visibilityScope = scope
}

function maybeStopVisibilityTracking() {
  if (visibilityScope === null || taskRegistry.size > 0) {
    return
  }

  visibilityScope.stop()
  visibilityScope = null
  // Nothing to reset: the next `ensureVisibilityTracking` rebinds `isDocumentVisible` to a
  // fresh vueuse ref that reads the live `visibilityState` before any task can read it.
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
