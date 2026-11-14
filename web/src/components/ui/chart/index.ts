import type { Component, Ref } from 'vue'
import { createContext } from 'reka-ui'

export { default as ChartContainer } from './ChartContainer.vue'
export { default as ChartLegendContent } from './ChartLegendContent.vue'
export { default as ChartTooltipContent } from './ChartTooltipContent.vue'
export { default as PillDonut } from './PillDonut.vue'
export { componentToString } from './utils'
export type { PillDonutDatum } from './pill-donut'

// Format: { THEME_NAME: CSS_SELECTOR }
export const THEMES = { light: '', dark: '.dark' } as const

type ChartConfigItem = {
  label?: string | Component
  icon?: string | Component
} & (
  | { color?: string; theme?: never }
  | { color?: never; theme: Record<keyof typeof THEMES, string> }
)

export type ChartConfig = Partial<Record<string, ChartConfigItem>>

interface ChartContextProps {
  id: string
  config: Ref<ChartConfig>
}

export const [useChart, provideChartContext] =
  createContext<ChartContextProps>('Chart')

export {
  VisCrosshair as ChartCrosshair,
  VisTooltip as ChartTooltip,
} from '@unovis/vue'
