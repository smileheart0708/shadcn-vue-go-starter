<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { TrendingDown, TrendingUp } from '@lucide/vue'

import SourceCard from '@/components/dashboard/SourceCard.vue'
import TrendCard from '@/components/dashboard/TrendCard.vue'
import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardAction,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

const { t } = useI18n()

// Demo dashboard tiles with static figures. Values stay hardcoded on purpose — swap
// this array for real data when the metrics endpoint exists. Copy is resolved here,
// not in the template, because `t()` keys must stay literal for the i18n lint rules.
const metrics = computed(() => [
  {
    id: 'totalRevenue',
    title: t('dashboard.metric.totalRevenue.title'),
    value: '$1,250.00',
    trend: '+12.5%',
    icon: TrendingUp,
  },
  {
    id: 'newCustomers',
    title: t('dashboard.metric.newCustomers.title'),
    value: '1,234',
    trend: '-20%',
    icon: TrendingDown,
  },
  {
    id: 'activeAccounts',
    title: t('dashboard.metric.activeAccounts.title'),
    value: '45,678',
    trend: '+12.5%',
    icon: TrendingUp,
  },
  {
    id: 'growthRate',
    title: t('dashboard.metric.growthRate.title'),
    value: '4.5%',
    trend: '+4.5%',
    icon: TrendingUp,
  },
])
</script>

<template>
  <!--
    fill 布局下页面高度是分配的，metrics 行越矮，下面两张图能用的高度越多。
    上下留白用 vh 的 clamp 压缩：大屏取到上限（与原来的 gap-4/gap-6、py-5 一致），
    只有视口变矮时才收缩，不需要额外的断点。
  -->
  <div
    class="@container/metrics flex flex-col gap-[clamp(0.75rem,1.8vh,1rem)] lg:flex-1 lg:gap-[clamp(1rem,2.7vh,1.5rem)] lg:min-block-0"
  >
    <!--
      列数按「容器宽度」而不是视口断点：侧栏是 offcanvas，收起后内容区会变宽，
      视口断点这时会误判。4 列要求每张至少 14rem（4×14+3×1 = 59rem），
      达不到就先掉到 2 列，再不够掉到 1 列 —— 不会出现 3+1 的孤行。
    -->
    <div
      class="grid grid-cols-1 gap-4 @[29rem]/metrics:grid-cols-2 @[59rem]/metrics:grid-cols-4"
    >
      <Card
        v-for="metric in metrics"
        :key="metric.id"
        class="@container/card py-[clamp(0.75rem,2.2vh,1.25rem)]"
      >
        <CardHeader>
          <CardDescription>{{ metric.title }}</CardDescription>
          <CardTitle
            class="text-2xl font-semibold tabular-nums @[250px]/card:text-3xl"
          >
            {{ metric.value }}
          </CardTitle>
          <CardAction>
            <Badge variant="outline">
              <component
                :is="metric.icon"
                class="block-4 inline-4"
              />
              {{ metric.trend }}
            </Badge>
          </CardAction>
        </CardHeader>
      </Card>
    </div>

    <div class="flex flex-1 flex-col gap-4 min-block-0 lg:grid lg:grid-cols-2">
      <TrendCard />
      <SourceCard />
    </div>
  </div>
</template>
