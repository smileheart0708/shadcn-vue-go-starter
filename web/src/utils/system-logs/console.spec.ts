import { describe, expect, it } from 'vitest'
import {
  createSystemLogStreamPresentation,
  reduceSystemLogStreamPresentation,
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
