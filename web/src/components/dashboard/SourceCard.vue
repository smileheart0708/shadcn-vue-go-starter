<script setup lang="ts">
import type { ChartLegendItem, DonutDatum } from '@/components/common/charts'
import { useResizeObserver } from '@vueuse/core'
import { computed, ref, useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ChartLegend,
  DONUT_COLORS,
  donutColorAt,
  PillDonut,
} from '@/components/common/charts'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

defineOptions({ name: 'DashboardSourceCard' })

const { t, locale } = useI18n()

// Demo breakdown, same as the metric tiles: replace with a real aggregate when the API exists.
const segments = computed<DonutDatum[]>(() => [
  { label: t('dashboard.trafficSources.segments.direct'), value: 4280 },
  { label: t('dashboard.trafficSources.segments.search'), value: 3120 },
  { label: t('dashboard.trafficSources.segments.social'), value: 1860 },
  { label: t('dashboard.trafficSources.segments.referral'), value: 1240 },
])

const total = computed(() =>
  segments.value.reduce((sum, segment) => sum + segment.value, 0),
)

const numberFormat = computed(() => new Intl.NumberFormat(locale.value))

const percentFormat = computed(
  () =>
    new Intl.NumberFormat(locale.value, {
      style: 'percent',
      maximumFractionDigits: 1,
    }),
)

const formattedTotal = computed(() => numberFormat.value.format(total.value))

/**
 * 图例色必须和环上的扇区色来自同一个函数：之前这里单独写了一遍
 * `DONUT_COLORS[index % length]`，一旦某段通过 `datum.color` 覆盖颜色，
 * 环和图例会显示两种颜色。统一走 donutColorAt 后只有一份取色规则。
 */
const legendItems = computed<ChartLegendItem[]>(() =>
  segments.value.map((segment, index) => ({
    key: segment.label,
    label: segment.label,
    color: donutColorAt(segment, index, DONUT_COLORS),
    value:
      total.value > 0
        ? percentFormat.value.format(segment.value / total.value)
        : '',
  })),
)

/**
 * 环形图跟卡片高度走，所以卡片被压矮时环会等比缩小。两处必须跟着高度调整：
 * 环宽按直径缩放，否则小环会被 28px 的粗细挤成四朵花；中心合计在环内放不下时
 * 直接隐藏，避免文字压到环上。两者都是按「可用高度」判断，宽度不参与。
 */
const donutArea = useTemplateRef<HTMLElement>('donutArea')
const donutAreaHeight = ref(0)

useResizeObserver(donutArea, (entries) => {
  donutAreaHeight.value = Math.round(entries[0]?.contentRect.height ?? 0)
})

const donutArcWidth = computed(() => {
  if (donutAreaHeight.value === 0) {
    return 28
  }

  return Math.min(28, Math.max(10, Math.round(donutAreaHeight.value * 0.12)))
})

const showDonutCenter = computed(
  () => donutAreaHeight.value === 0 || donutAreaHeight.value >= 200,
)
</script>

<template>
  <Card class="flex-col gap-0 overflow-hidden py-0 block-full min-block-0">
    <CardHeader class="border-be py-5">
      <CardTitle>{{ t('dashboard.trafficSources.title') }}</CardTitle>
      <CardDescription>
        {{ t('dashboard.trafficSources.description') }}
      </CardDescription>
    </CardHeader>
    <CardContent class="flex flex-1 flex-col gap-5 px-6 py-5 min-block-0">
      <!--
        高度跟卡片走：桌面端 flex 撑满，矮视口自动缩小；`max-*` 只限制大屏下的
        环直径，不要再写死 aspect-square，否则卡片会被固定高度顶出可视区。
        窄屏（单列、页面可滚动）用 min-block 给出确定高度，供 unovis 首帧测量。
      -->
      <div
        ref="donutArea"
        class="flex flex-1 items-center justify-center min-block-52 lg:min-block-0"
      >
        <PillDonut
          :data="segments"
          :arc-width="donutArcWidth"
          class="max-block-60 max-inline-60"
        >
          <template #center>
            <div
              v-if="showDonutCenter"
              class="grid gap-1 text-center"
            >
              <span class="text-2xl font-semibold tabular-nums">
                {{ formattedTotal }}
              </span>
              <span class="text-xs text-muted-foreground">
                {{ t('dashboard.trafficSources.total') }}
              </span>
            </div>
          </template>
        </PillDonut>
      </div>
      <ChartLegend
        :items="legendItems"
        class="grid grid-cols-2 gap-x-6 gap-y-2.5"
      />
    </CardContent>
  </Card>
</template>
