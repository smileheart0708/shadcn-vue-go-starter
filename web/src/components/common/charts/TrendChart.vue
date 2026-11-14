<script setup lang="ts" generic="TDatum extends Record<string, unknown>">
import type { ChartLegendItem, ChartSeries, TooltipSeries } from './chart.ts'
import { useResizeObserver } from '@vueuse/core'
import { computed, onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'
import {
  VisArea,
  VisAxis,
  VisCrosshair,
  VisTooltip,
  VisXYContainer,
} from '@unovis/vue'
import { cn } from '@/lib/utils'
import ChartContainer from './ChartContainer.vue'
import ChartLegend from './ChartLegend.vue'
import ChartTooltipContent from './ChartTooltipContent.vue'
import { componentToString } from './chart.ts'

const props = withDefaults(
  defineProps<{
    data: TDatum[]
    series: readonly ChartSeries<TDatum>[]
    /** X 轴取值，返回时间戳或 Date。 */
    x: (datum: TDatum) => number | Date
    /** X 轴刻度文案。 */
    xTickFormat: (value: number) => string
    /** tooltip 标题文案。 */
    xTooltipFormat: (value: number | Date) => string
    xNumTicks?: number
    /**
     * 宿主布局动画（侧栏开合、面板收展）开始/结束时翻转的布尔值。
     * 翻转期间冻结图表尺寸，避免 unovis 跟着连续宽度重排造成抖动。
     * 图表层不认识侧栏，只消费这个信号，才能在没有 SidebarProvider 的布局里复用。
     */
    freezeOnLayoutChange?: boolean
    /** 冻结时长，毫秒；由宿主按自己的过渡时长传入。 */
    freezeDuration?: number
    class?: string
  }>(),
  {
    xNumTicks: 6,
    freezeDuration: 220,
    class: undefined,
  },
)

const yAccessors = computed(() => props.series.map((item) => item.value))

const legendItems = computed<ChartLegendItem[]>(() =>
  props.series.map((item) => ({
    key: item.key,
    label: item.label,
    color: item.color,
  })),
)

const tooltipSeries = computed<Record<string, TooltipSeries>>(() => {
  const entries: [string, TooltipSeries][] = props.series.map((item) => [
    item.key,
    { label: item.label, color: item.color },
  ])

  return Object.fromEntries(entries)
})

const crosshairTemplate = computed(() =>
  componentToString(tooltipSeries.value, ChartTooltipContent, {
    labelFormatter: props.xTooltipFormat,
  }),
)

/**
 * 面积、折线与十字准线共用一套系列色，避免三处 accessor 各自维护一份。
 * unovis 只会按系列下标调用，取模后必然落在 series 内。
 */
function colorAccessor(_datum: TDatum, index: number): string {
  const series = props.series
  return series.length > 0
    ? series[index % series.length].color
    : 'currentColor'
}

/**
 * Y 轴左边距按数值量级估算：刻度文案宽度随位数变化，固定留白会让四位数被裁。
 */
const chartMargin = computed(() => {
  let maxValue = 0

  for (const datum of props.data) {
    for (const item of props.series) {
      maxValue = Math.max(maxValue, item.value(datum))
    }
  }

  const labelWidth = maxValue.toLocaleString().length

  return {
    left: Math.min(Math.max(40, labelWidth * 8 + 12), 80),
    right: 8,
    top: 12,
    bottom: 28,
  }
})

let freezeTimer: number | null = null
const isChartFrozen = ref(false)
const pendingChartWidth = ref<number | null>(null)
const chartShell = useTemplateRef<HTMLElement>('chartShell')
const chartWidth = ref(0)

useResizeObserver(chartShell, (entries) => {
  const nextWidth = Math.round(entries[0]?.contentRect.width ?? 0)
  if (nextWidth <= 0) {
    return
  }

  if (isChartFrozen.value) {
    pendingChartWidth.value = nextWidth
    return
  }

  chartWidth.value = nextWidth
  pendingChartWidth.value = null
})

const clearFreezeTimer = () => {
  if (freezeTimer !== null) {
    window.clearTimeout(freezeTimer)
    freezeTimer = null
  }
}

/**
 * 宿主布局动画会让容器宽度连续变化，unovis 每次都重算布局，表现为折线抖动。
 * 动画期间把 XY 容器钉在旧尺寸，动画结束后再一次性交回真实宽度。
 * 时长由 `freezeDuration` 传入，图表层不猜宿主用了多久的过渡。
 */
const freezeChartForLayoutChange = () => {
  const shell = chartShell.value
  if (!shell) {
    return
  }

  const visContainer = shell.querySelector<HTMLElement>(
    '[data-vis-xy-container]',
  )
  if (!visContainer) {
    return
  }

  const { width, height } = visContainer.getBoundingClientRect()
  if (!width || !height) {
    return
  }

  visContainer.style.width = String(width) + 'px'
  visContainer.style.height = String(height) + 'px'
  shell.style.overflow = 'hidden'
  isChartFrozen.value = true

  clearFreezeTimer()
  freezeTimer = window.setTimeout(() => {
    visContainer.style.removeProperty('width')
    visContainer.style.removeProperty('height')
    shell.style.removeProperty('overflow')
    isChartFrozen.value = false
    if (pendingChartWidth.value !== null) {
      chartWidth.value = pendingChartWidth.value
      pendingChartWidth.value = null
    }
  }, props.freezeDuration)
}

watch(
  () => props.freezeOnLayoutChange,
  () => {
    freezeChartForLayoutChange()
  },
  { flush: 'sync' },
)

onBeforeUnmount(() => {
  clearFreezeTimer()
})
</script>

<template>
  <div
    ref="chartShell"
    :class="
      cn('relative flex flex-1 flex-col block-full min-block-0', props.class)
    "
  >
    <ChartContainer
      v-if="props.data.length > 0 && chartWidth > 0"
      class="flex-1 min-block-0"
    >
      <VisXYContainer
        :data="props.data"
        :width="chartWidth"
        :margin="chartMargin"
        :auto-margin="false"
        class="flex-1 block-auto! min-block-0"
      >
        <VisArea
          :x="props.x"
          :y="yAccessors"
          :color="colorAccessor"
          :opacity="0.4"
          :line="true"
          :line-width="1"
        />
        <VisAxis
          type="x"
          :x="props.x"
          :tick-line="false"
          :domain-line="false"
          :grid-line="false"
          :num-ticks="props.xNumTicks"
          :tick-format="props.xTickFormat"
        />
        <VisAxis
          type="y"
          :num-ticks="4"
          :tick-line="false"
          :domain-line="false"
        />
        <VisTooltip />
        <VisCrosshair
          :template="crosshairTemplate"
          :color="colorAccessor"
        />
      </VisXYContainer>

      <ChartLegend
        :items="legendItems"
        class="gap-4 pbs-3"
      />
    </ChartContainer>
  </div>
</template>
