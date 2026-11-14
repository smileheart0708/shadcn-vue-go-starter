import type { Component } from 'vue'
import { isClient } from '@vueuse/core'
import { useId } from 'reka-ui'
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
 * crosshair 的 HTML 片段完全由数据推导，随时可以重建，所以缓存不需要淘汰策略，
 * 超过上限直接清空即可。没有上限的话，切换语言会重建 series、生成新的实例 id，
 * 旧命名空间下的每条悬浮内容都会永久留在内存里。
 */
const CACHE_LIMIT = 256
const cache = new Map<string, string>()

function serializeKey(key: Record<string, unknown>): string {
  return JSON.stringify(key, Object.keys(key).sort())
}

/**
 * unovis 的 crosshair/tooltip 只接受渲染后的 HTML 字符串，所以这里把 Vue 组件
 * 挂到游离节点上取 innerHTML。crosshair 每次移动都会调用，按「实例 id + payload」
 * 缓存可以避免同一段数据反复创建 vnode。
 */
export function componentToString(
  series: Record<string, TooltipSeries>,
  component: Component,
  props: Record<string, unknown> = {},
) {
  if (!isClient) return

  const id = useId()

  return (rawData: unknown, x: number | Date) => {
    const data = getChartPayload(rawData)
    const serializedKey = `${id}-${serializeKey(data)}`
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
