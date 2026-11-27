import { describe, expect, it } from 'vitest'
import {
  createSystemLogStreamPresentation,
  reduceSystemLogStreamPresentation,
  selectSystemLogEntryWindow,
  SYSTEM_LOG_ENTRY_ROW_PITCH,
  type SystemLogStreamEvent,
  type SystemLogStreamPresentation,
} from './console'

function applyEvents(
  events: readonly SystemLogStreamEvent[],
): SystemLogStreamPresentation {
  let state = createSystemLogStreamPresentation()
  for (const event of events) {
    state = reduceSystemLogStreamPresentation(state, event)
  }
  return state
}

describe('system log stream presentation', () => {
  it('starts in the connecting/loading state', () => {
    expect(createSystemLogStreamPresentation()).toEqual({
      phase: 'connecting',
      error: '',
      loading: true,
    })
  })

  it('keeps one stable disconnected view while a flapping stream retries', () => {
    let state = applyEvents([
      { type: 'start' },
      { type: 'failed', message: 'connection reset' },
    ])

    const snapshots: SystemLogStreamPresentation[] = []
    for (let attempt = 0; attempt < 6; attempt += 1) {
      state = reduceSystemLogStreamPresentation(state, {
        type: 'failed',
        message: attempt % 2 === 0 ? '503' : 'network error',
      })
      snapshots.push(state)
    }

    expect(
      new Set(
        snapshots.map(
          (snapshot) =>
            `${snapshot.phase}|${snapshot.error}|${String(snapshot.loading)}`,
        ),
      ).size,
    ).toBe(1)
    expect(state).toEqual({
      phase: 'disconnected',
      error: 'connection reset',
      loading: false,
    })
  })

  it('shows entries without claiming the connection is stable', () => {
    const state = applyEvents([
      { type: 'start' },
      { type: 'entry' },
      { type: 'failed', message: 'stream closed' },
    ])

    expect(state).toEqual({
      phase: 'disconnected',
      error: 'stream closed',
      loading: false,
    })
  })

  it('publishes connected and clears the error only on the stable event', () => {
    const disconnected = applyEvents([
      { type: 'start' },
      { type: 'failed', message: 'stream unavailable' },
      { type: 'failed', message: 'stream unavailable' },
    ])

    const otherEvents: SystemLogStreamEvent[] = [
      { type: 'start' },
      { type: 'entry' },
      { type: 'failed', message: 'stream unavailable' },
      { type: 'fatal', message: 'stream unavailable' },
    ]
    for (const event of otherEvents) {
      expect(
        reduceSystemLogStreamPresentation(disconnected, event).phase,
      ).not.toBe('connected')
    }

    expect(
      reduceSystemLogStreamPresentation(disconnected, { type: 'stable' }),
    ).toEqual({ phase: 'connected', error: '', loading: false })
  })

  it('a terminal failure replaces the message kept from retries', () => {
    const retried = applyEvents([
      { type: 'start' },
      { type: 'failed', message: '503 service unavailable' },
      { type: 'failed', message: 'network error' },
    ])

    expect(
      reduceSystemLogStreamPresentation(retried, {
        type: 'fatal',
        message: 'forbidden',
      }),
    ).toEqual({ phase: 'disconnected', error: 'forbidden', loading: false })
  })

  it('a fresh session shows connecting without restoring the skeleton list', () => {
    const loaded = applyEvents([
      { type: 'start' },
      { type: 'entry' },
      { type: 'stable' },
      { type: 'failed', message: 'stream unavailable' },
    ])

    expect(
      reduceSystemLogStreamPresentation(loaded, { type: 'start' }),
    ).toEqual({ phase: 'connecting', error: '', loading: false })
  })
})

describe('system log entry window', () => {
  const viewportHeight = 512
  const visibleRowCount = Math.ceil(viewportHeight / SYSTEM_LOG_ENTRY_ROW_PITCH)

  it('mounts only the first screen plus overscan at the top', () => {
    const entryWindow = selectSystemLogEntryWindow({
      entryCount: 10_000,
      scrollTop: 0,
      viewportHeight,
    })

    expect(entryWindow.start).toBe(0)
    expect(entryWindow.end).toBeGreaterThan(visibleRowCount)
    expect(entryWindow.end).toBeLessThan(40)
  })

  it('follows the scroll position instead of mounting the whole buffer', () => {
    const firstVisibleIndex = 5_000
    const entryWindow = selectSystemLogEntryWindow({
      entryCount: 10_000,
      scrollTop: firstVisibleIndex * SYSTEM_LOG_ENTRY_ROW_PITCH,
      viewportHeight,
    })

    expect(entryWindow.start).toBeLessThanOrEqual(firstVisibleIndex)
    expect(entryWindow.end).toBeGreaterThanOrEqual(
      firstVisibleIndex + visibleRowCount,
    )
    expect(entryWindow.end - entryWindow.start).toBeLessThan(40)
  })

  it('clamps the window to the end of the buffer', () => {
    const entryCount = 100
    const entryWindow = selectSystemLogEntryWindow({
      entryCount,
      scrollTop: entryCount * SYSTEM_LOG_ENTRY_ROW_PITCH - viewportHeight,
      viewportHeight,
    })

    expect(entryWindow.end).toBe(entryCount)
    expect(entryWindow.start).toBeLessThan(entryWindow.end)
  })

  it('treats a scroll position past the end as the bottom', () => {
    expect(
      selectSystemLogEntryWindow({
        entryCount: 10,
        scrollTop: 10_000,
        viewportHeight,
      }),
    ).toEqual({ start: 0, end: 10 })
  })

  it('returns an empty window for an empty buffer', () => {
    expect(
      selectSystemLogEntryWindow({
        entryCount: 0,
        scrollTop: 0,
        viewportHeight,
      }),
    ).toEqual({ start: 0, end: 0 })
  })
})
