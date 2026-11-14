<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

/**
 * 图表容器只做两件事：撑满父级、重置 unovis 的浮层与坐标轴样式。
 * 高度由调用方决定（桌面端 flex 撑满、窄屏给固定高度），所以这里不再写
 * aspect-video —— 那条固定比例会让每个调用方都要用 class 覆盖回来。
 *
 * 十字准线只承担 tooltip 的定位，不画竖线（线宽固定 0）。需要显式画线时再加
 * 开关，不要恢复成一个没有调用方的布尔 prop。
 */
const props = defineProps<{
  class?: HTMLAttributes['class']
}>()

defineSlots<{ default: () => unknown }>()
</script>

<template>
  <div
    data-slot="chart"
    :class="
      cn(
        'flex flex-col justify-center text-xs block-full inline-full [&_.tick_line]:stroke-border/50! [&_.tick_text]:fill-muted-foreground!',
        props.class,
      )
    "
    :style="{
      '--vis-tooltip-padding': '0px',
      '--vis-tooltip-background-color': 'transparent',
      '--vis-tooltip-border-color': 'transparent',
      '--vis-tooltip-text-color': 'none',
      '--vis-tooltip-shadow-color': 'none',
      '--vis-tooltip-backdrop-filter': 'none',
      '--vis-crosshair-circle-stroke-color': '#0000',
      '--vis-crosshair-line-stroke-width': '0px',
      '--vis-font-family': 'var(--font-sans)',
    }"
  >
    <slot />
  </div>
</template>
