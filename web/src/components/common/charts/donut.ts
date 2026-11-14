export interface DonutDatum {
  readonly label: string
  readonly value: number
  readonly color?: string
}

/**
 * 至少包含一个颜色。空数组会让 `index % 0` 得到 `NaN`、取色变成 `undefined`，
 * 所以把「非空」写进类型，让调用方在编译期就被拦住，而不是在运行时回退兜底色。
 */
export type DonutPalette = readonly [string, ...string[]]

export function hasVisibleValue(data: readonly DonutDatum[]): boolean {
  return data.some((datum) => datum.value > 0)
}

/**
 * 索引来自 unovis 的系列下标，一定是非负整数，取模后必然落在调色板内。
 * 色板只在 palette.ts 维护一份，这里不再准备兜底颜色。
 */
export function donutColorAt(
  datum: DonutDatum,
  index: number,
  colors: DonutPalette,
): string {
  if (datum.color !== undefined) return datum.color
  return colors[index % colors.length]
}

export function describeDonut(data: readonly DonutDatum[]): string {
  return data
    .map((datum) => `${datum.label}: ${String(datum.value)}`)
    .join(', ')
}
