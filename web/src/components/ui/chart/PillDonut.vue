<script setup lang="ts">
/**
 * 本组件直接挂 VisSingleContainer，没有复用同目录的 ChartContainer：
 * ChartContainer 强制 `aspect-video`，并通过 `--vis-*` 变量与 ChartStyle 承担
 * 容器撑满和主题色注入；药丸环形图要固定像素高度，且没有坐标轴、浮层和 SVG 文字，
 * 那套主题变量一个都用不上。不要为了「统一」把它塞进 ChartContainer，
 * 否则要么丢掉高度契约，要么被迫复制一份 ChartContainer 的样式覆盖。
 */
import type { PillDonutDatum } from './pill-donut'
import { VisDonut, VisSingleContainer } from '@unovis/vue'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import {
  describePillDonut,
  hasVisibleValue,
  resolveDatumColor,
} from './pill-donut'

interface Props {
  data: readonly PillDonutDatum[]
  colors?: readonly string[]
  /** 环宽，像素。 */
  arcWidth?: number
  /** 段间间隙，弧度。 */
  gap?: number
  /** 端部圆角，默认取环宽的一半，保证端头是完整半圆。 */
  cornerRadius?: number
  /** 数据更新与移除的过渡时长，毫秒。 */
  duration?: number
  /** 入场扫开时长，毫秒。 */
  enterDuration?: number
  /** 组件高度，像素。宽度始终撑满父容器。 */
  height?: number
  ariaLabel?: string
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  colors: () => [],
  arcWidth: 28,
  gap: 0.07,
  duration: 800,
  enterDuration: 700,
  height: 260,
  ariaLabel: undefined,
  cornerRadius: undefined,
  class: undefined,
})

defineOptions({ name: 'PillDonut' })

defineSlots<{ center?: () => unknown }>()

const displayData = ref<PillDonutDatum[]>([])
const entering = ref(true)

let timerId = 0

/**
 * `immediate` 取代单独的 onBeforeMount 初始化：两者都在首次渲染前同步赋值，
 * 但只有 watch 会在后续 props 变化时继续同步。
 * `deep` 是为了兼容调用方原地改数组元素；对外契约仍是只读，复制一份避免
 * unovis 持有调用方的数组引用。
 */
watch(
  () => props.data,
  (next) => {
    displayData.value = next.map((datum) => ({ ...datum }))
  },
  { immediate: true, deep: true },
)

const rendered = computed(() => hasVisibleValue(displayData.value))

/**
 * 入场扫开动画跟随「首次真正画出扇区」的时机，而不是组件挂载时机。
 * 图表由 `rendered` 控制挂载：数据异步到达、或首屏数值全是 0 时，挂载那一刻还
 * 没有可绘制的扇区。若按挂载计时，遮罩会在图表出现之前被摘掉，动画永远不播。
 * 因此这里等 `rendered` 首次为真才开始计时，并且只播一次，之后的数据更新交给
 * unovis 自己的角度过渡。
 */
watch(
  rendered,
  (visible) => {
    if (!visible || !entering.value) return
    timerId = window.setTimeout(() => {
      entering.value = false
    }, props.enterDuration + 60)
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  window.clearTimeout(timerId)
})

const resolvedCornerRadius = computed(
  () => props.cornerRadius ?? props.arcWidth / 2,
)
const resolvedAriaLabel = computed(
  () => props.ariaLabel ?? describePillDonut(props.data),
)
const rootStyle = computed(() => ({
  height: `${String(props.height)}px`,
  '--pill-donut-enter-duration': `${String(props.enterDuration)}ms`,
}))

function valueAccessor(datum: PillDonutDatum): number {
  return datum.value
}

function colorAccessor(datum: PillDonutDatum, index: number): string {
  return resolveDatumColor(datum, index, props.colors)
}
</script>

<template>
  <div
    data-slot="pill-donut"
    :data-entering="entering ? '' : undefined"
    :class="props.class"
    :style="rootStyle"
    role="img"
    :aria-label="resolvedAriaLabel"
  >
    <VisSingleContainer
      v-if="rendered"
      data-slot="pill-donut-chart"
      :data="displayData"
    >
      <VisDonut
        :value="valueAccessor"
        :color="colorAccessor"
        :arc-width="arcWidth"
        :pad-angle="gap"
        :corner-radius="resolvedCornerRadius"
        :duration="duration"
        :show-background="false"
      />
    </VisSingleContainer>
    <div
      v-if="$slots.center"
      data-slot="pill-donut-center"
    >
      <slot name="center" />
    </div>
  </div>
</template>

<style scoped>
@property --pill-donut-sweep {
  syntax: '<angle>';
  inherits: false;
  initial-value: 0deg;
}

[data-slot='pill-donut'] {
  position: relative;
  width: 100%;
}

[data-slot='pill-donut-chart'] {
  width: 100%;
  height: 100%;
}

[data-slot='pill-donut'][data-entering] [data-slot='pill-donut-chart'] {
  -webkit-mask-image: conic-gradient(
    #000 var(--pill-donut-sweep),
    transparent var(--pill-donut-sweep)
  );
  mask-image: conic-gradient(
    #000 var(--pill-donut-sweep),
    transparent var(--pill-donut-sweep)
  );
  animation: pill-donut-sweep var(--pill-donut-enter-duration, 700ms)
    cubic-bezier(0.22, 1, 0.36, 1) both;
}

[data-slot='pill-donut-center'] {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

@keyframes pill-donut-sweep {
  from {
    --pill-donut-sweep: 0deg;
  }

  to {
    --pill-donut-sweep: 360deg;
  }
}
</style>
