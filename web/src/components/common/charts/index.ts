/**
 * 图表层唯一的对外入口。这里只导出页面真正消费的东西：
 * `ChartContainer`、`ChartTooltipContent`、`componentToString` 是 `TrendChart` 的
 * 内部实现，不对外暴露，否则会出现「直接用容器拼图」的第二套用法，两套并存后
 * 样式与交互必然漂移。需要新图表时，在层内组合后再从这里导出。
 * 移除本层可一并清理 `@unovis/*` 与 `pnpm-workspace.yaml` 的图表依赖项。
 */
export { default as ChartLegend } from './ChartLegend.vue'
export { default as PillDonut } from './PillDonut.vue'
export { default as TrendChart } from './TrendChart.vue'
export { DONUT_COLORS, TREND_COLORS } from './palette.ts'
export { donutColorAt } from './donut.ts'
export type { ChartLegendItem, ChartSeries } from './chart.ts'
export type { DonutDatum } from './donut.ts'
