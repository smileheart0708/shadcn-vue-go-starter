<script setup lang="ts">
import type { TooltipSeries } from './chart'
import { computed } from 'vue'

/**
 * 由 componentToString 渲染到游离节点上，不在 ChartContainer 的子树里，
 * 所以这里只能用 props 拿系列信息，不能依赖容器注入的 context。
 * 组件只在图表层内部使用（不经 index.ts 导出），因此不提供 class 之类的
 * 外部样式入口，避免出现第二套使用方式。
 */
const props = withDefaults(
  defineProps<{
    payload?: Record<string, unknown>
    series?: Record<string, TooltipSeries>
    x?: number | Date
    labelFormatter?: (value: number | Date) => string
    /** 由 TrendChart 透传；图表层不读 i18n。 */
    locale: string
  }>(),
  {
    payload: () => ({}),
    series: () => ({}),
  },
)

interface TooltipItem {
  key: string
  label: string
  color: string
  value: string
}

const tooltipLabel = computed<string | null>(() => {
  if (props.labelFormatter !== undefined && props.x !== undefined) {
    return props.labelFormatter(props.x)
  }

  return props.x === undefined ? null : String(props.x)
})

const payloadItems = computed<TooltipItem[]>(() => {
  const items: TooltipItem[] = []

  for (const [key, value] of Object.entries(props.payload)) {
    if (!Object.hasOwn(props.series, key)) {
      continue
    }

    const series = props.series[key]

    items.push({
      key,
      label: series.label,
      color: series.color,
      value: formatTooltipValue(value),
    })
  }

  return items
})

function formatTooltipValue(value: unknown): string {
  if (value === null || value === undefined) {
    return ''
  }

  if (typeof value === 'string') {
    return value
  }

  if (typeof value === 'number' || typeof value === 'bigint') {
    return value.toLocaleString(props.locale)
  }

  if (value instanceof Date) {
    return value.toLocaleString(props.locale)
  }

  if (typeof value === 'object') {
    return JSON.stringify(value)
  }

  if (typeof value === 'boolean') {
    return value ? 'true' : 'false'
  }

  return ''
}
</script>

<template>
  <div
    class="grid items-start gap-1.5 rounded-lg border border-border/50 bg-background px-2.5 py-1.5 text-xs shadow-xl min-inline-32"
  >
    <div
      v-if="tooltipLabel !== null"
      class="font-medium"
    >
      {{ tooltipLabel }}
    </div>
    <div class="grid gap-1.5">
      <div
        v-for="item in payloadItems"
        :key="item.key"
        class="flex items-center gap-2 inline-full"
      >
        <div
          class="shrink-0 rounded-[2px] block-2.5 inline-2.5"
          :style="{ backgroundColor: item.color }"
        />
        <div class="flex flex-1 items-center justify-between leading-none">
          <span class="text-muted-foreground">{{ item.label }}</span>
          <span class="font-mono font-medium text-foreground tabular-nums">
            {{ item.value }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
