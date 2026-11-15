<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import type { ChartLegendItem } from './chart'
import { cn } from '@/lib/utils'

/**
 * 折线与环形图共用一份图例，所以它只接收 items，不读容器 context：
 * 环形图挂在 VisSingleContainer 上，拿不到 XY 容器的选择器。
 * 排列方式（横排、两列网格）由调用方通过 class 决定，这里不预设间距。
 */
const props = defineProps<{
  items: readonly ChartLegendItem[]
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <ul :class="cn('flex items-center justify-center text-xs', props.class)">
    <li
      v-for="item in props.items"
      :key="item.key"
      class="flex items-center gap-1.5"
    >
      <span
        class="shrink-0 rounded-[2px] block-2 inline-2"
        :style="{ backgroundColor: item.color }"
      />
      <span>{{ item.label }}</span>
      <span
        v-if="item.value !== undefined"
        class="ms-auto font-medium tabular-nums"
      >
        {{ item.value }}
      </span>
    </li>
  </ul>
</template>
