/**
 * 图表色源按「图表类型」分组，与 style.css 里的 `--chart-<type>-*` 一一对应。
 * 不要跨组复用：折线与环形共用一套色板时，调整其中一种图的观感会连带改到另一种。
 * 新增一类图表就新增一组变量和这里的一行导出。
 */
export const TREND_COLORS = [
  'var(--chart-line-1)',
  'var(--chart-line-2)',
] as const

export const DONUT_COLORS = [
  'var(--chart-donut-1)',
  'var(--chart-donut-2)',
  'var(--chart-donut-3)',
  'var(--chart-donut-4)',
  'var(--chart-donut-5)',
] as const
