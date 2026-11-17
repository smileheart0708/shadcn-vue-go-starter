import { describe, expect, it } from 'vitest'
import { h } from 'vue'
import { componentToString } from './chart'

interface ReviewTooltipProps {
  payload: Record<string, unknown>
  series: Record<string, { label: string }>
  x: number
  labelFormatter: (value: number) => string
}

function ReviewTooltip(props: ReviewTooltipProps) {
  return h(
    'div',
    `${props.labelFormatter(props.x)}|${props.series.desktop.label}|${String(props.payload.desktop)}`,
  )
}

function series(label: string) {
  return { desktop: { label, color: '#000' } }
}

const payload = { desktop: 1 }

describe('componentToString', () => {
  it('keeps tooltip html per series instead of sharing one cache namespace', () => {
    const first = componentToString(series('Alpha'), ReviewTooltip, {
      locale: 'en-US',
      labelFormatter: String,
    })
    const second = componentToString(series('Beta'), ReviewTooltip, {
      locale: 'en-US',
      labelFormatter: String,
    })

    expect(first?.(payload, 1)).toContain('Alpha')
    expect(second?.(payload, 1)).toContain('Beta')
  })

  it('keys the formatted label so a new formatter is not served stale html', () => {
    const first = componentToString(series('Desktop'), ReviewTooltip, {
      locale: 'en-US',
      labelFormatter: () => 'first',
    })
    const second = componentToString(series('Desktop'), ReviewTooltip, {
      locale: 'en-US',
      labelFormatter: () => 'second',
    })

    expect(first?.(payload, 1)).toContain('first')
    expect(second?.(payload, 1)).toContain('second')
  })
})
