export interface PillDonutDatum {
  readonly label: string
  readonly value: number
  readonly color?: string
}

/**
 * 默认调色板优先取药丸环形图专属变量，变量缺失时退回内置十六进制色。
 * 第二个参数是给「脱离本项目的复用」准备的：style.css 已经定义了 `--pill-donut-*`，
 * 在本仓库里 fallback 永远不会生效，值由 `:root` / `.dark` 决定。
 */
export const PILL_DONUT_DEFAULT_COLORS: readonly string[] = [
  'var(--pill-donut-1, #6366f1)',
  'var(--pill-donut-2, #22d3ee)',
  'var(--pill-donut-3, #34d399)',
  'var(--pill-donut-4, #fbbf24)',
  'var(--pill-donut-5, #f472b6)',
]

export function hasVisibleValue(data: readonly PillDonutDatum[]): boolean {
  return data.some((datum) => datum.value > 0)
}

/**
 * 索引来自 unovis 的系列下标，一定是非负整数，取模后必然落在调色板内，
 * 所以不额外维护兜底颜色；调色板为空时回落到默认色板。
 */
export function resolveDatumColor(
  datum: PillDonutDatum,
  index: number,
  colors: readonly string[],
): string {
  if (datum.color !== undefined) return datum.color

  const palette = colors.length > 0 ? colors : PILL_DONUT_DEFAULT_COLORS
  return palette[index % palette.length]
}

export function describePillDonut(data: readonly PillDonutDatum[]): string {
  return data
    .map((datum) => `${datum.label}: ${String(datum.value)}`)
    .join(', ')
}
