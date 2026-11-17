import type { Component } from 'vue'
import { isClient } from '@vueuse/core'
import { h, render } from 'vue'

/**
 * 系列是图表层唯一的对外契约：折线的轴、颜色与提示浮层，环形图的图例与分段色
 * 都从这一份数据派生，调用方不再各自维护颜色映射。
 */
export interface ChartSeries<TDatum> {
  readonly key: string
  readonly label: string
  readonly color: string
  readonly value: (datum: TDatum) => number
}

export interface ChartLegendItem {
  readonly key: string
  readonly label: string
  readonly color: string
  /** 图例右侧的附加值，环形图用它显示占比。 */
  readonly value?: string
}

export interface TooltipSeries {
  readonly label: string
  readonly color: string
}

/**
 * crosshair 的 HTML 片段只由渲染输入决定：payload、series、x、locale，以及标题格式化后
 * 的文案。缓存键因此必须覆盖这些输入，而不是图表实例——`useId()` 在 computed 里拿不到
 * 组件实例，会退化成常量，多图同页时第二个图表会复用第一个的 HTML。函数进不了 JSON
 * 键，所以用 labelFormatter 的输出参与键；否则只换格式化函数、locale 不变时会命中旧内容。
 * 片段随时可以重建，所以缓存不需要淘汰策略，超过上限直接清空即可。
 */
const CACHE_LIMIT = 256
const cache = new Map<string, string>()

type LabelFormatter = (value: number | Date) => string

function isLabelFormatter(value: unknown): value is LabelFormatter {
  return typeof value === 'function'
}

/**
 * 稳定深序列化：对象键排序后再交给 JSON，避免同一份数据因属性插入顺序不同而漏命中；
 * 函数与 undefined 由 JSON.stringify 自行忽略。
 */
function serializeKey(value: unknown): string {
  return JSON.stringify(value, (_key: string, item: unknown): unknown => {
    if (!isRecord(item) || Array.isArray(item)) {
      return item
    }

    return Object.fromEntries(
      Object.entries(item).sort(([left], [right]) => left.localeCompare(right)),
    )
  })
}

/**
 * unovis 的 crosshair/tooltip 只接受渲染后的 HTML 字符串，所以这里把 Vue 组件
 * 挂到游离节点上取 innerHTML。crosshair 每次移动都会调用，按「渲染输入」缓存可以
 * 避免同一段数据反复创建 vnode。
 */
export function componentToString(
  series: Record<string, TooltipSeries>,
  component: Component,
  props: Record<string, unknown> = {},
) {
  if (!isClient) return

  return (rawData: unknown, x: number | Date) => {
    const data = getChartPayload(rawData)
    const labelFormatter = props.labelFormatter
    // 函数无法进 JSON 键：用格式化结果参与键，保证换格式化函数时也不复用旧 HTML。
    const label = isLabelFormatter(labelFormatter)
      ? labelFormatter(x)
      : undefined
    const serializedKey = serializeKey({ ...props, label, x, series, data })
    const cachedContent = cache.get(serializedKey)
    if (cachedContent !== undefined) return cachedContent

    const vnode = h<unknown>(component, { ...props, payload: data, series, x })
    const div = document.createElement('div')
    render(vnode, div)
    if (cache.size >= CACHE_LIMIT) {
      cache.clear()
    }
    cache.set(serializedKey, div.innerHTML)
    return div.innerHTML
  }
}

function getChartPayload(value: unknown): Record<string, unknown> {
  if (!isRecord(value)) {
    return {}
  }

  if (isRecord(value.data)) {
    return value.data
  }

  return value
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
