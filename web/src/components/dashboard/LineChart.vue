<script setup lang="ts">
import type { ChartConfig } from '@/components/ui/chart'
import { useResizeObserver } from '@vueuse/core'
import { computed, onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'
import { useSidebar } from '@/components/ui/sidebar'
import { VisArea, VisAxis, VisLine, VisXYContainer } from '@unovis/vue'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  ChartContainer,
  ChartCrosshair,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  componentToString,
} from '@/components/ui/chart'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

defineOptions({ name: 'DashboardLineChart' })

/**
 * Demo chart with static data — the only consumer of `@unovis` in the app.
 *
 * COST: unovis pulls in maplibre-gl transitively (hence the maplibre-gl / stylus /
 * vue-metamorph pins in pnpm-workspace.yaml) and makes DashboardView the largest chunk
 * in the bundle by a wide margin. It is lazily loaded with the route, so it costs
 * nothing until the dashboard is opened.
 *
 * If you are not shipping charts, deleting this component plus `components/ui/chart/`
 * lets you drop both `@unovis/*` dependencies and all three pnpm overrides. If you are,
 * replace `chartData` with a real series and keep the rest.
 */
const chartData = [
  { date: new Date('2024-04-01'), desktop: 222, mobile: 150 },
  { date: new Date('2024-04-02'), desktop: 97, mobile: 180 },
  { date: new Date('2024-04-03'), desktop: 167, mobile: 120 },
  { date: new Date('2024-04-04'), desktop: 242, mobile: 260 },
  { date: new Date('2024-04-05'), desktop: 373, mobile: 290 },
  { date: new Date('2024-04-06'), desktop: 301, mobile: 340 },
  { date: new Date('2024-04-07'), desktop: 245, mobile: 180 },
  { date: new Date('2024-04-08'), desktop: 409, mobile: 320 },
  { date: new Date('2024-04-09'), desktop: 59, mobile: 110 },
  { date: new Date('2024-04-10'), desktop: 261, mobile: 190 },
  { date: new Date('2024-04-11'), desktop: 327, mobile: 350 },
  { date: new Date('2024-04-12'), desktop: 292, mobile: 210 },
  { date: new Date('2024-04-13'), desktop: 342, mobile: 380 },
  { date: new Date('2024-04-14'), desktop: 137, mobile: 220 },
  { date: new Date('2024-04-15'), desktop: 120, mobile: 170 },
  { date: new Date('2024-04-16'), desktop: 138, mobile: 190 },
  { date: new Date('2024-04-17'), desktop: 446, mobile: 360 },
  { date: new Date('2024-04-18'), desktop: 364, mobile: 410 },
  { date: new Date('2024-04-19'), desktop: 243, mobile: 180 },
  { date: new Date('2024-04-20'), desktop: 89, mobile: 150 },
  { date: new Date('2024-04-21'), desktop: 137, mobile: 200 },
  { date: new Date('2024-04-22'), desktop: 224, mobile: 170 },
  { date: new Date('2024-04-23'), desktop: 138, mobile: 230 },
  { date: new Date('2024-04-24'), desktop: 387, mobile: 290 },
  { date: new Date('2024-04-25'), desktop: 215, mobile: 250 },
  { date: new Date('2024-04-26'), desktop: 75, mobile: 130 },
  { date: new Date('2024-04-27'), desktop: 383, mobile: 420 },
  { date: new Date('2024-04-28'), desktop: 122, mobile: 180 },
  { date: new Date('2024-04-29'), desktop: 315, mobile: 240 },
  { date: new Date('2024-04-30'), desktop: 454, mobile: 380 },
  { date: new Date('2024-05-01'), desktop: 165, mobile: 220 },
  { date: new Date('2024-05-02'), desktop: 293, mobile: 310 },
  { date: new Date('2024-05-03'), desktop: 247, mobile: 190 },
  { date: new Date('2024-05-04'), desktop: 385, mobile: 420 },
  { date: new Date('2024-05-05'), desktop: 481, mobile: 390 },
  { date: new Date('2024-05-06'), desktop: 498, mobile: 520 },
  { date: new Date('2024-05-07'), desktop: 388, mobile: 300 },
  { date: new Date('2024-05-08'), desktop: 149, mobile: 210 },
  { date: new Date('2024-05-09'), desktop: 227, mobile: 180 },
  { date: new Date('2024-05-10'), desktop: 293, mobile: 330 },
  { date: new Date('2024-05-11'), desktop: 335, mobile: 270 },
  { date: new Date('2024-05-12'), desktop: 197, mobile: 240 },
  { date: new Date('2024-05-13'), desktop: 197, mobile: 160 },
  { date: new Date('2024-05-14'), desktop: 448, mobile: 490 },
  { date: new Date('2024-05-15'), desktop: 473, mobile: 380 },
  { date: new Date('2024-05-16'), desktop: 338, mobile: 400 },
  { date: new Date('2024-05-17'), desktop: 499, mobile: 420 },
  { date: new Date('2024-05-18'), desktop: 315, mobile: 350 },
  { date: new Date('2024-05-19'), desktop: 235, mobile: 180 },
  { date: new Date('2024-05-20'), desktop: 177, mobile: 230 },
  { date: new Date('2024-05-21'), desktop: 82, mobile: 140 },
  { date: new Date('2024-05-22'), desktop: 81, mobile: 120 },
  { date: new Date('2024-05-23'), desktop: 252, mobile: 290 },
  { date: new Date('2024-05-24'), desktop: 294, mobile: 220 },
  { date: new Date('2024-05-25'), desktop: 201, mobile: 250 },
  { date: new Date('2024-05-26'), desktop: 213, mobile: 170 },
  { date: new Date('2024-05-27'), desktop: 420, mobile: 460 },
  { date: new Date('2024-05-28'), desktop: 233, mobile: 190 },
  { date: new Date('2024-05-29'), desktop: 78, mobile: 130 },
  { date: new Date('2024-05-30'), desktop: 340, mobile: 280 },
  { date: new Date('2024-05-31'), desktop: 178, mobile: 230 },
  { date: new Date('2024-06-01'), desktop: 178, mobile: 200 },
  { date: new Date('2024-06-02'), desktop: 470, mobile: 410 },
  { date: new Date('2024-06-03'), desktop: 103, mobile: 160 },
  { date: new Date('2024-06-04'), desktop: 439, mobile: 380 },
  { date: new Date('2024-06-05'), desktop: 88, mobile: 140 },
  { date: new Date('2024-06-06'), desktop: 294, mobile: 250 },
  { date: new Date('2024-06-07'), desktop: 323, mobile: 370 },
  { date: new Date('2024-06-08'), desktop: 385, mobile: 320 },
  { date: new Date('2024-06-09'), desktop: 438, mobile: 480 },
  { date: new Date('2024-06-10'), desktop: 155, mobile: 200 },
  { date: new Date('2024-06-11'), desktop: 92, mobile: 150 },
  { date: new Date('2024-06-12'), desktop: 492, mobile: 420 },
  { date: new Date('2024-06-13'), desktop: 81, mobile: 130 },
  { date: new Date('2024-06-14'), desktop: 426, mobile: 380 },
  { date: new Date('2024-06-15'), desktop: 307, mobile: 350 },
  { date: new Date('2024-06-16'), desktop: 371, mobile: 310 },
  { date: new Date('2024-06-17'), desktop: 475, mobile: 520 },
  { date: new Date('2024-06-18'), desktop: 107, mobile: 170 },
  { date: new Date('2024-06-19'), desktop: 341, mobile: 290 },
  { date: new Date('2024-06-20'), desktop: 408, mobile: 450 },
  { date: new Date('2024-06-21'), desktop: 169, mobile: 210 },
  { date: new Date('2024-06-22'), desktop: 317, mobile: 270 },
  { date: new Date('2024-06-23'), desktop: 480, mobile: 530 },
  { date: new Date('2024-06-24'), desktop: 132, mobile: 180 },
  { date: new Date('2024-06-25'), desktop: 141, mobile: 190 },
  { date: new Date('2024-06-26'), desktop: 434, mobile: 380 },
  { date: new Date('2024-06-27'), desktop: 448, mobile: 490 },
  { date: new Date('2024-06-28'), desktop: 149, mobile: 200 },
  { date: new Date('2024-06-29'), desktop: 103, mobile: 160 },
  { date: new Date('2024-06-30'), desktop: 446, mobile: 400 },
] as const

type Data = (typeof chartData)[number]
type TimeRange = '90d' | '30d' | '7d'

const chartConfig = {
  mobile: {
    label: 'Mobile',
    color: 'var(--chart-2)',
  },
  desktop: {
    label: 'Desktop',
    color: 'var(--chart-5)',
  },
} satisfies ChartConfig

const timeRange = ref<TimeRange>('90d')
const chartWidth = ref(0)
const displayedRange = computed(() => timeRange.value)

const filteredChartData = computed<Data[]>(() => {
  const referenceDate = new Date('2024-06-30')
  let daysToSubtract = 90

  if (timeRange.value === '30d') {
    daysToSubtract = 30
  } else if (timeRange.value === '7d') {
    daysToSubtract = 7
  }

  const startDate = new Date(referenceDate)
  startDate.setDate(startDate.getDate() - daysToSubtract)

  return chartData.filter((item) => item.date >= startDate)
})

const chartMargin = computed(() => {
  const maxValue = filteredChartData.value.reduce(
    (max, point) => Math.max(max, point.mobile, point.desktop),
    0,
  )
  const yLabelWidth = maxValue.toLocaleString().length

  return {
    left: Math.min(Math.max(40, yLabelWidth * 8 + 12), 80),
    right: 8,
    top: 12,
    bottom: 28,
  }
})

const { open } = useSidebar()
const sidebarAnimationMs = 220
let freezeTimer: number | null = null
const isChartFrozen = ref(false)
const pendingChartWidth = ref<number | null>(null)
const chartShell = useTemplateRef<HTMLElement>('chartShell')

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

const freezeChartDuringSidebarAnimation = () => {
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
  }, sidebarAnimationMs)
}

watch(
  open,
  () => {
    freezeChartDuringSidebarAnimation()
  },
  { flush: 'sync' },
)

onBeforeUnmount(() => {
  clearFreezeTimer()
})

function formatAxisDate(value: number) {
  return new Date(value).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
  })
}

function formatTooltipDate(value: number | Date) {
  return new Date(value).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}
</script>

<template>
  <Card class="flex-col gap-0 overflow-hidden py-0 block-full min-block-0">
    <CardHeader
      class="flex items-center gap-2 space-y-0 border-be py-5 sm:flex-row"
    >
      <div class="grid flex-1 gap-1">
        <CardTitle>Area Chart - Interactive</CardTitle>
        <CardDescription
          >Showing total visitors for the last 3 months</CardDescription
        >
      </div>
      <Select v-model="timeRange">
        <SelectTrigger
          class="hidden rounded-lg inline-40 sm:ms-auto sm:flex"
          aria-label="Select a value"
        >
          <SelectValue placeholder="Last 3 months" />
        </SelectTrigger>
        <SelectContent class="rounded-xl">
          <SelectItem
            value="90d"
            class="rounded-lg"
          >
            Last 3 months
          </SelectItem>
          <SelectItem
            value="30d"
            class="rounded-lg"
          >
            Last 30 days
          </SelectItem>
          <SelectItem
            value="7d"
            class="rounded-lg"
          >
            Last 7 days
          </SelectItem>
        </SelectContent>
      </Select>
    </CardHeader>
    <CardContent class="flex-1 px-2 py-4 min-block-0 sm:px-6 sm:pbs-6">
      <div
        ref="chartShell"
        class="relative flex flex-col block-full min-block-0"
      >
        <div class="relative flex flex-1 block-full min-block-0">
          <ChartContainer
            v-if="filteredChartData.length > 0 && chartWidth > 0"
            :config="chartConfig"
            class="opacity-100 transition-[opacity,filter] duration-200 block-full inline-full min-block-0"
            :cursor="false"
          >
            <VisXYContainer
              :data="filteredChartData"
              :width="chartWidth > 0 ? chartWidth : undefined"
              :margin="chartMargin"
              :auto-margin="false"
              class="flex-1 block-auto! min-block-0"
            >
              <VisArea
                :x="(d: Data) => d.date"
                :y="[(d: Data) => d.mobile, (d: Data) => d.desktop]"
                :color="
                  (datum: Data, index: number) => {
                    void datum
                    return (
                      [chartConfig.mobile.color, chartConfig.desktop.color][
                        index
                      ] ?? 'var(--chart-1)'
                    )
                  }
                "
                :opacity="0.4"
              />
              <VisLine
                :x="(d: Data) => d.date"
                :y="[(d: Data) => d.mobile, (d: Data) => d.mobile + d.desktop]"
                :color="
                  (datum: Data, index: number) => {
                    void datum
                    return (
                      [chartConfig.mobile.color, chartConfig.desktop.color][
                        index
                      ] ?? 'var(--chart-1)'
                    )
                  }
                "
                :line-width="1"
              />
              <VisAxis
                type="x"
                :x="(d: Data) => d.date"
                :tick-line="false"
                :domain-line="false"
                :grid-line="false"
                :num-ticks="displayedRange === '7d' ? 7 : 6"
                :tick-format="formatAxisDate"
              />
              <VisAxis
                type="y"
                :num-ticks="4"
                :tick-line="false"
                :domain-line="false"
              />
              <ChartTooltip />
              <ChartCrosshair
                :template="
                  componentToString(chartConfig, ChartTooltipContent, {
                    labelFormatter: formatTooltipDate,
                  })
                "
                :color="
                  (datum: Data, index: number) => {
                    void datum
                    return (
                      [chartConfig.mobile.color, chartConfig.desktop.color][
                        index % 2
                      ] ?? 'var(--chart-1)'
                    )
                  }
                "
              />
            </VisXYContainer>

            <ChartLegendContent />
          </ChartContainer>
        </div>
      </div>
    </CardContent>
  </Card>
</template>
