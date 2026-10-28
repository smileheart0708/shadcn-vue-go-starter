<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ChartArea, Clock, TrendingDown, TrendingUp } from '@lucide/vue'

import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import LineChart from '@/components/dashboard/LineChart.vue'

const { t } = useI18n()

// Demo dashboard tiles with static figures. Values stay hardcoded on purpose — swap
// this array for real data when the metrics endpoint exists; the copy is keyed by
// metric id under `dashboard.metric.*`.
const metrics = [
  { id: 'totalRevenue', value: '$1,250.00', trend: '+12.5%', icon: TrendingUp },
  { id: 'newCustomers', value: '1,234', trend: '-20%', icon: TrendingDown },
  { id: 'activeAccounts', value: '45,678', trend: '+12.5%', icon: TrendingUp },
  { id: 'growthRate', value: '4.5%', trend: '+4.5%', icon: TrendingUp },
] as const
</script>

<template>
  <div class="flex flex-col gap-4 lg:flex-1 lg:gap-6 lg:min-block-0">
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <Card
        v-for="metric in metrics"
        :key="metric.id"
        class="@container/card gap-4 py-5"
      >
        <CardHeader class="pbe-0">
          <CardDescription>{{
            t(`dashboard.metric.${metric.id}.title`)
          }}</CardDescription>
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
        <CardFooter class="flex-col items-start gap-1.5 pbs-0 text-sm">
          <div class="line-clamp-1 flex gap-2 font-medium">
            {{ t(`dashboard.metric.${metric.id}.summary`) }}
            <component
              :is="metric.icon"
              class="block-4 inline-4"
            />
          </div>
          <div class="text-muted-foreground">
            {{ t(`dashboard.metric.${metric.id}.detail`) }}
          </div>
        </CardFooter>
      </Card>
    </div>

    <div class="flex flex-1 flex-col gap-4 min-block-0 lg:grid lg:grid-cols-2">
      <LineChart />

      <Card class="gap-0 overflow-hidden py-5 min-block-0 min-inline-0">
        <CardHeader class="shrink-0 pbe-2">
          <CardDescription>{{
            t('dashboard.reservedPanel.eyebrow')
          }}</CardDescription>
          <CardTitle class="text-xl font-semibold">{{
            t('dashboard.reservedPanel.title')
          }}</CardTitle>
          <CardAction>
            <Badge variant="outline">
              <Clock class="block-4 inline-4" />
              {{ t('dashboard.reservedPanel.badge') }}
            </Badge>
          </CardAction>
        </CardHeader>
        <CardContent class="flex flex-1 min-block-0 min-inline-0">
          <Empty class="border-border/70 bg-muted/30 block-full min-inline-0">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <ChartArea class="block-5 inline-5" />
              </EmptyMedia>
              <EmptyTitle>{{
                t('dashboard.reservedPanel.emptyTitle')
              }}</EmptyTitle>
              <EmptyDescription>
                {{ t('dashboard.reservedPanel.emptyDescription') }}
              </EmptyDescription>
            </EmptyHeader>
            <EmptyContent class="max-inline-none min-inline-0">
              <div class="grid gap-3 inline-full">
                <div
                  class="flex items-center justify-between gap-3 rounded-lg border bg-background/80 px-4 py-3 min-inline-0"
                >
                  <span class="text-muted-foreground">{{
                    t('dashboard.reservedPanel.suggestedWidth')
                  }}</span>
                  <span class="truncate text-end font-medium">{{
                    t('dashboard.reservedPanel.suggestedWidthValue')
                  }}</span>
                </div>
                <div
                  class="flex items-center justify-between gap-3 rounded-lg border bg-background/80 px-4 py-3 min-inline-0"
                >
                  <span class="text-muted-foreground">{{
                    t('dashboard.reservedPanel.suggestedContent')
                  }}</span>
                  <span class="truncate text-end font-medium">{{
                    t('dashboard.reservedPanel.suggestedContentValue')
                  }}</span>
                </div>
                <div
                  class="flex items-center justify-between gap-3 rounded-lg border bg-background/80 px-4 py-3 min-inline-0"
                >
                  <span class="text-muted-foreground">{{
                    t('dashboard.reservedPanel.viewportRule')
                  }}</span>
                  <span class="truncate text-end font-medium">{{
                    t('dashboard.reservedPanel.viewportRuleValue')
                  }}</span>
                </div>
              </div>
            </EmptyContent>
          </Empty>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
