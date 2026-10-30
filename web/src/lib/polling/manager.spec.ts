import { nextTick, ref } from 'vue'
import {
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from 'vitest'
import {
  acquirePollingTask,
  type PollingTaskOptions,
  type PollingTaskSubscription,
} from '@/lib/polling/manager'

type FetchStub = PollingTaskOptions<number>['fetch']
type ApplyStub = PollingTaskOptions<number>['apply']

// `no-empty-function` is on for this project, and an empty stub has nothing to say anyway.
const noopApply: ApplyStub = () => undefined

const acquiredTasks: PollingTaskSubscription[] = []

function acquireTask(
  options: PollingTaskOptions<number>,
): PollingTaskSubscription {
  const task = acquirePollingTask(options)
  acquiredTasks.push(task)
  return task
}

function createDeferred<TData>() {
  let settle: (value: TData) => void = () => undefined

  const promise = new Promise<TData>((resolve) => {
    settle = resolve
  })

  return {
    promise,
    resolve(value: TData) {
      settle(value)
    },
  }
}

let visibilityState: DocumentVisibilityState = 'visible'

/** Mirrors a tab switch: vueuse reacts to the event, so the assertion must wait a tick. */
async function setVisibility(next: DocumentVisibilityState) {
  visibilityState = next
  document.dispatchEvent(new Event('visibilitychange'))
  await nextTick()
}

beforeAll(() => {
  // jsdom exposes `visibilityState` as a prototype getter; the suite needs to drive it.
  Object.defineProperty(document, 'visibilityState', {
    configurable: true,
    get: () => visibilityState,
  })
})

beforeEach(() => {
  visibilityState = 'visible'
  vi.useFakeTimers()
})

afterEach(() => {
  for (const task of acquiredTasks) {
    task.release()
  }

  acquiredTasks.length = 0
  vi.restoreAllMocks()
  vi.useRealTimers()
})

describe('polling task scheduling', () => {
  it('waits one interval before the first run', async () => {
    const fetchData = vi.fn<FetchStub>().mockResolvedValue(1)

    const task = acquireTask({
      key: 'scheduling.first-run',
      intervalMs: 1000,
      fetch: fetchData,
      apply: noopApply,
    })

    task.start()

    await vi.advanceTimersByTimeAsync(999)
    expect(fetchData).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1)
    expect(fetchData).toHaveBeenCalledTimes(1)
  })

  it('shares a single timer between consumers of the same key', async () => {
    const firstFetch = vi.fn<FetchStub>().mockResolvedValue(1)
    const secondFetch = vi.fn<FetchStub>().mockResolvedValue(2)

    const first = acquireTask({
      key: 'scheduling.shared-key',
      intervalMs: 1000,
      fetch: firstFetch,
      apply: noopApply,
    })
    const second = acquireTask({
      key: 'scheduling.shared-key',
      intervalMs: 1000,
      fetch: secondFetch,
      apply: noopApply,
    })

    first.start()
    second.start()

    await vi.advanceTimersByTimeAsync(1000)

    // One request for both consumers, and it is the newest subscriber's `fetch`: that is
    // the documented price of sharing a key, so a drifting second implementation fails here.
    expect(secondFetch).toHaveBeenCalledTimes(1)
    expect(firstFetch).not.toHaveBeenCalled()

    // One released consumer must not tear down the timer the other still needs.
    first.release()

    await vi.advanceTimersByTimeAsync(1000)
    expect(secondFetch).toHaveBeenCalledTimes(2)

    second.release()

    await vi.advanceTimersByTimeAsync(10_000)
    expect(secondFetch).toHaveBeenCalledTimes(2)
  })

  it('re-arms the next tick when intervalMs changes', async () => {
    const intervalMs = ref(1000)
    const fetchData = vi.fn<FetchStub>().mockResolvedValue(1)

    const task = acquireTask({
      key: 'scheduling.interval-change',
      intervalMs,
      fetch: fetchData,
      apply: noopApply,
    })

    task.start()

    await vi.advanceTimersByTimeAsync(1000)
    expect(fetchData).toHaveBeenCalledTimes(1)

    intervalMs.value = 3000
    await nextTick()

    await vi.advanceTimersByTimeAsync(2999)
    expect(fetchData).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(1)
    expect(fetchData).toHaveBeenCalledTimes(2)
  })

  it('applies a new interval that arrives while a request is in flight', async () => {
    const intervalMs = ref(1000)
    const pending = createDeferred<number>()
    const fetchData = vi.fn<FetchStub>(() => pending.promise)

    const task = acquireTask({
      key: 'scheduling.interval-change-in-flight',
      intervalMs,
      fetch: fetchData,
      apply: noopApply,
    })

    task.start()

    await vi.advanceTimersByTimeAsync(1000)
    expect(fetchData).toHaveBeenCalledTimes(1)

    intervalMs.value = 5000
    await nextTick()

    // The delay is recomputed when the run settles, so the change must neither run a request
    // of its own nor make the settle fall back to the old 1000ms cadence.
    pending.resolve(1)
    await vi.advanceTimersByTimeAsync(4999)
    expect(fetchData).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(1)
    expect(fetchData).toHaveBeenCalledTimes(2)
  })
})

describe('polling task manual runs', () => {
  it('honours refreshNow issued before the task starts', async () => {
    const fetchData = vi.fn<FetchStub>().mockResolvedValue(1)

    const task = acquireTask({
      key: 'manual.before-start',
      intervalMs: 1000,
      fetch: fetchData,
      apply: noopApply,
    })

    task.refreshNow()
    expect(fetchData).not.toHaveBeenCalled()

    task.start()
    await vi.advanceTimersByTimeAsync(0)
    expect(fetchData).toHaveBeenCalledTimes(1)
  })

  it('coalesces refreshNow while a request is in flight', async () => {
    const pending = createDeferred<number>()
    const fetchData = vi.fn<FetchStub>(() => pending.promise)

    const task = acquireTask({
      key: 'manual.in-flight',
      intervalMs: 1000,
      fetch: fetchData,
      apply: noopApply,
    })

    task.start()
    task.refreshNow()

    await vi.advanceTimersByTimeAsync(0)
    expect(fetchData).toHaveBeenCalledTimes(1)
    expect(task.refreshing.value).toBe(true)

    task.refreshNow()

    await vi.advanceTimersByTimeAsync(0)
    expect(fetchData).toHaveBeenCalledTimes(1)

    pending.resolve(1)
    await vi.advanceTimersByTimeAsync(0)

    // The request that arrived mid-flight runs once the previous one settles, never in
    // parallel with it.
    expect(fetchData).toHaveBeenCalledTimes(2)
  })
})

describe('polling task document visibility', () => {
  it('pauses while hidden and refreshes once on return', async () => {
    const fetchData = vi.fn<FetchStub>().mockResolvedValue(1)

    const task = acquireTask({
      key: 'visibility.pause',
      intervalMs: 1000,
      fetch: fetchData,
      apply: noopApply,
    })

    task.start()

    await vi.advanceTimersByTimeAsync(1000)
    expect(fetchData).toHaveBeenCalledTimes(1)

    await setVisibility('hidden')
    expect(task.paused.value).toBe(true)

    await vi.advanceTimersByTimeAsync(10_000)
    expect(fetchData).toHaveBeenCalledTimes(1)

    await setVisibility('visible')
    expect(task.paused.value).toBe(false)

    await vi.advanceTimersByTimeAsync(0)
    expect(fetchData).toHaveBeenCalledTimes(2)
  })

  it('aborts the in-flight request when the document becomes hidden', async () => {
    const pending = createDeferred<number>()
    const signals: AbortSignal[] = []
    const fetchData = vi.fn<FetchStub>(({ signal }) => {
      signals.push(signal)
      return pending.promise
    })

    const task = acquireTask({
      key: 'visibility.abort',
      intervalMs: 1000,
      fetch: fetchData,
      apply: noopApply,
    })

    task.start()

    await vi.advanceTimersByTimeAsync(1000)
    expect(signals).toHaveLength(1)
    expect(signals[0]?.aborted).toBe(false)

    await setVisibility('hidden')
    expect(signals[0]?.aborted).toBe(true)
    expect(task.refreshing.value).toBe(false)

    pending.resolve(1)
    await vi.advanceTimersByTimeAsync(10_000)

    // An aborted run is neither a success nor a failure, so it must not re-arm a timer.
    expect(fetchData).toHaveBeenCalledTimes(1)
  })

  it('re-reads visibility when tracking restarts after the last release', async () => {
    // No task exists yet, so this only moves `document.visibilityState`; the tracking that
    // the next task installs has to read the live value instead of a remembered one.
    await setVisibility('hidden')

    const idle = acquireTask({
      key: 'visibility.restart',
      intervalMs: 1000,
      fetch: vi.fn<FetchStub>().mockResolvedValue(1),
      apply: noopApply,
    })
    idle.release()

    const fetchData = vi.fn<FetchStub>().mockResolvedValue(1)
    const restarted = acquireTask({
      key: 'visibility.restart',
      intervalMs: 1000,
      fetch: fetchData,
      apply: noopApply,
    })

    restarted.start()
    expect(restarted.paused.value).toBe(true)

    await vi.advanceTimersByTimeAsync(10_000)
    expect(fetchData).not.toHaveBeenCalled()

    await setVisibility('visible')
    expect(restarted.paused.value).toBe(false)

    await vi.advanceTimersByTimeAsync(0)
    expect(fetchData).toHaveBeenCalledTimes(1)
  })
})

describe('polling task failure handling', () => {
  it('backs off exponentially after failures and resets on success', async () => {
    let shouldFail = true
    const fetchData = vi.fn<FetchStub>(() =>
      shouldFail ? Promise.reject(new Error('boom')) : Promise.resolve(1),
    )

    const task = acquireTask({
      key: 'failure.backoff',
      intervalMs: 1000,
      fetch: fetchData,
      apply: noopApply,
    })

    task.start()

    await vi.advanceTimersByTimeAsync(1000)
    expect(fetchData).toHaveBeenCalledTimes(1)
    expect(task.consecutiveFailures.value).toBe(1)

    // First failure retries after 1x the interval.
    await vi.advanceTimersByTimeAsync(1000)
    expect(fetchData).toHaveBeenCalledTimes(2)
    expect(task.consecutiveFailures.value).toBe(2)

    // Second failure retries after 2x the interval.
    await vi.advanceTimersByTimeAsync(1999)
    expect(fetchData).toHaveBeenCalledTimes(2)

    await vi.advanceTimersByTimeAsync(1)
    expect(fetchData).toHaveBeenCalledTimes(3)
    expect(task.consecutiveFailures.value).toBe(3)

    // Third failure retries after 4x, and this attempt succeeds.
    shouldFail = false

    await vi.advanceTimersByTimeAsync(4000)
    expect(fetchData).toHaveBeenCalledTimes(4)
    expect(task.consecutiveFailures.value).toBe(0)

    // A success restores the configured cadence instead of the backoff delay.
    await vi.advanceTimersByTimeAsync(1000)
    expect(fetchData).toHaveBeenCalledTimes(5)
  })

  it('caps the backoff at maxBackoffMs', async () => {
    const fetchData = vi.fn<FetchStub>().mockRejectedValue(new Error('down'))

    const task = acquireTask({
      key: 'failure.cap',
      intervalMs: 1000,
      maxBackoffMs: 3000,
      fetch: fetchData,
      apply: noopApply,
    })

    task.start()

    await vi.advanceTimersByTimeAsync(1000)
    await vi.advanceTimersByTimeAsync(1000)
    await vi.advanceTimersByTimeAsync(2000)

    expect(fetchData).toHaveBeenCalledTimes(3)

    // 4x the interval would be 4000ms, but the cap keeps it at 3000ms.
    await vi.advanceTimersByTimeAsync(2999)
    expect(fetchData).toHaveBeenCalledTimes(3)

    await vi.advanceTimersByTimeAsync(1)
    expect(fetchData).toHaveBeenCalledTimes(4)
  })
})
