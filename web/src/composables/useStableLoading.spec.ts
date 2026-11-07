import { effectScope, nextTick, ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  LOADING_MIN_VISIBLE_MS,
  LOADING_SHOW_DELAY_MS,
  useStableLoading,
} from './useStableLoading'

function mountStableLoading(initial: boolean) {
  const pending = ref(initial)
  const scope = effectScope()
  const state = scope.run(() => useStableLoading(pending))
  if (state === undefined) {
    throw new Error('useStableLoading returned no state.')
  }

  return { pending, scope, ...state }
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useStableLoading', () => {
  it('does not paint a skeleton when the first response arrives quickly', async () => {
    const { pending, scope, showLoading } = mountStableLoading(true)
    await nextTick()

    await vi.advanceTimersByTimeAsync(LOADING_SHOW_DELAY_MS - 10)
    pending.value = false
    await nextTick()
    await vi.advanceTimersByTimeAsync(500)

    expect(showLoading.value).toBe(false)
    scope.stop()
  })

  it('shows the skeleton only after the delay and keeps it for the minimum duration', async () => {
    const { pending, scope, showLoading } = mountStableLoading(true)
    await nextTick()

    await vi.advanceTimersByTimeAsync(LOADING_SHOW_DELAY_MS - 1)
    expect(showLoading.value).toBe(false)
    await vi.advanceTimersByTimeAsync(1)
    expect(showLoading.value).toBe(true)

    pending.value = false
    await nextTick()
    expect(showLoading.value).toBe(true)
    await vi.advanceTimersByTimeAsync(LOADING_MIN_VISIBLE_MS - 1)
    expect(showLoading.value).toBe(true)
    await vi.advanceTimersByTimeAsync(1)
    expect(showLoading.value).toBe(false)

    scope.stop()
  })

  it('keeps the skeleton visible across a request that starts again', async () => {
    const { pending, scope, showLoading } = mountStableLoading(true)
    await nextTick()
    await vi.advanceTimersByTimeAsync(LOADING_SHOW_DELAY_MS)

    pending.value = false
    await nextTick()
    pending.value = true
    await nextTick()
    await vi.advanceTimersByTimeAsync(LOADING_MIN_VISIBLE_MS * 2)

    expect(showLoading.value).toBe(true)
    scope.stop()
  })

  it('clears pending timers when the owning scope stops', async () => {
    const { pending, scope, showLoading } = mountStableLoading(true)
    await nextTick()
    await vi.advanceTimersByTimeAsync(LOADING_SHOW_DELAY_MS)
    expect(showLoading.value).toBe(true)

    pending.value = false
    await nextTick()
    scope.stop()
    await vi.advanceTimersByTimeAsync(LOADING_MIN_VISIBLE_MS * 2)

    expect(showLoading.value).toBe(true)
  })

  it('withholds the first paint instead of letting consumers render an empty state', async () => {
    const { scope, showLoading, firstLoadPending } = mountStableLoading(true)
    await nextTick()

    expect(showLoading.value).toBe(false)
    expect(firstLoadPending.value).toBe(true)

    await vi.advanceTimersByTimeAsync(LOADING_SHOW_DELAY_MS)

    // 骨架已出现，就不再是「首屏待返回」：消费方此时该画骨架。
    expect(showLoading.value).toBe(true)
    expect(firstLoadPending.value).toBe(false)

    scope.stop()
  })

  it('stops withholding once the first response settles', async () => {
    const { pending, scope, showLoading, firstLoadPending } =
      mountStableLoading(true)
    await nextTick()

    await vi.advanceTimersByTimeAsync(20)
    pending.value = false
    await nextTick()

    expect(showLoading.value).toBe(false)
    // 首个结果已到，此时「没有数据」是真没有数据，空态可以显示。
    expect(firstLoadPending.value).toBe(false)
    scope.stop()
  })

  it('never withholds when the page starts without pending work', async () => {
    const { scope, showLoading, firstLoadPending } = mountStableLoading(false)
    await nextTick()

    expect(showLoading.value).toBe(false)
    expect(firstLoadPending.value).toBe(false)
    scope.stop()
  })

  it('keeps rendered content on a reload instead of blanking it', async () => {
    const { pending, scope, showLoading, firstLoadPending } =
      mountStableLoading(true)
    await nextTick()
    pending.value = false
    await nextTick()
    expect(firstLoadPending.value).toBe(false)

    pending.value = true
    await nextTick()
    await vi.advanceTimersByTimeAsync(LOADING_SHOW_DELAY_MS - 1)

    // 抑制窗口仍然生效（先不画骨架），但内容区保持原样，不能退回首屏留白。
    expect(showLoading.value).toBe(false)
    expect(firstLoadPending.value).toBe(false)

    await vi.advanceTimersByTimeAsync(1)
    expect(showLoading.value).toBe(true)
    scope.stop()
  })
})
