<script setup lang="ts">
import type { ChartSeries } from '@/components/common/charts'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { TrendChart, TREND_COLORS } from '@/components/common/charts'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useSidebar } from '@/components/ui/sidebar'

defineOptions({ name: 'DashboardTrendCard' })

/**
 * Demo dashboard chart with static data. Swap `chartData` for a real series when the
 * metrics endpoint exists; the chart layer itself carries no business copy.
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
]

type Data = (typeof chartData)[number]

const TIME_RANGES = ['90d', '30d', '7d'] as const
type TimeRange = (typeof TIME_RANGES)[number]

const { t, locale } = useI18n()

const { open: sidebarOpen } = useSidebar()

/**
 * 侧栏展开/收起的过渡时长。图表层只消费 `freezeDuration`，不认识侧栏，
 * 所以外壳动画若调整，只需要改这一个数字，不会出现图表冻结窗口与动画对不上的漂移。
 */
const SIDEBAR_TRANSITION_MS = 220

const series = computed<readonly ChartSeries<Data>[]>(() => [
  {
    key: 'mobile',
    label: t('dashboard.chart.series.mobile'),
    color: TREND_COLORS[0],
    value: (datum) => datum.mobile,
  },
  {
    key: 'desktop',
    label: t('dashboard.chart.series.desktop'),
    color: TREND_COLORS[1],
    value: (datum) => datum.desktop,
  },
])

const timeRange = ref<TimeRange>('90d')

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

function xValue(datum: Data): Date {
  return datum.date
}

function formatAxisDate(value: number): string {
  return new Date(value).toLocaleDateString(locale.value, {
    month: 'short',
    day: 'numeric',
  })
}

function formatTooltipDate(value: number | Date): string {
  return new Date(value).toLocaleDateString(locale.value, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

// 穷尽 switch：i18n lint 只认 t() 调用里的字面量键。
function rangeLabel(range: TimeRange): string {
  switch (range) {
    case '90d':
      return t('dashboard.chart.ranges.90d')
    case '30d':
      return t('dashboard.chart.ranges.30d')
    case '7d':
      return t('dashboard.chart.ranges.7d')
  }
}
</script>

<template>
  <Card class="flex-col gap-0 overflow-hidden py-0 block-full min-block-0">
    <CardHeader
      class="flex items-center gap-2 space-y-0 border-be py-5 sm:flex-row"
    >
      <div class="grid flex-1 gap-1">
        <CardTitle>{{ t('dashboard.chart.title') }}</CardTitle>
        <CardDescription>
          {{ t('dashboard.chart.description') }}
        </CardDescription>
      </div>
      <Select v-model="timeRange">
        <SelectTrigger
          class="hidden rounded-lg inline-40 sm:ms-auto sm:flex"
          :aria-label="t('dashboard.chart.rangeLabel')"
        >
          <SelectValue :placeholder="t('dashboard.chart.rangeLabel')" />
        </SelectTrigger>
        <SelectContent class="rounded-xl">
          <SelectItem
            v-for="range in TIME_RANGES"
            :key="range"
            :value="range"
            class="rounded-lg"
          >
            {{ rangeLabel(range) }}
          </SelectItem>
        </SelectContent>
      </Select>
    </CardHeader>
    <CardContent class="flex-1 px-2 py-4 min-block-0 sm:px-6 sm:pbs-6">
      <TrendChart
        :data="filteredChartData"
        :series="series"
        :x="xValue"
        :x-tick-format="formatAxisDate"
        :x-tooltip-format="formatTooltipDate"
        :x-num-ticks="timeRange === '7d' ? 7 : 6"
        :freeze-on-layout-change="sidebarOpen"
        :freeze-duration="SIDEBAR_TRANSITION_MS"
      />
    </CardContent>
  </Card>
</template>
