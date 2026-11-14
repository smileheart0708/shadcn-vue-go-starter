<script setup lang="ts">
/**
 * 本组件直接挂 VisSingleContainer，没有复用同目录的 ChartContainer：
 * ChartContainer 承担的是 XY 图表的 `--vis-*` 浮层与坐标轴样式，环形图既没有
 * 坐标轴也没有十字准线，套进去只会复制一份用不上的样式覆盖。
 * 高度交由父容器决定（父级必须是确定高度或 flex 撑满），组件本身不再写死像素。
 *
 * 对外只保留 `data / colors / arcWidth / class`：环宽要跟随卡片高度变化，色板要和
 * 图例共用同一份。间隙、圆角、过渡时长没有调用方需要调整，直接内联成常量，
 * 避免维护一批永远走默认值的分支；将来确有调用方时再加 prop。
 */
import type { DonutDatum, DonutPalette } from './donut'
import { VisDonut, VisSingleContainer } from '@unovis/vue'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { describeDonut, donutColorAt, hasVisibleValue } from './donut'
import { DONUT_COLORS } from './palette'

interface Props {
  data: readonly DonutDatum[]
  colors?: DonutPalette
  /** 环宽，像素；由调用方按可用高度缩放。 */
  arcWidth?: number
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  colors: () => DONUT_COLORS,
  arcWidth: 28,
  class: undefined,
})

/** 段间间隙，弧度。 */
const ARC_GAP = 0.07
/** 数据更新与移除的过渡时长，毫秒。 */
const UPDATE_DURATION_MS = 800
/** 入场扫开时长，毫秒。 */
const ENTER_DURATION_MS = 700

defineOptions({ name: 'PillDonut' })

defineSlots<{ center?: () => unknown }>()

const displayData = ref<DonutDatum[]>([])
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
    }, ENTER_DURATION_MS + 60)
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  window.clearTimeout(timerId)
})

const resolvedAriaLabel = computed(() => describeDonut(props.data))

function valueAccessor(datum: DonutDatum): number {
  return datum.value
}

function colorAccessor(datum: DonutDatum, index: number): string {
  return donutColorAt(datum, index, props.colors)
}
</script>

<template>
  <div
    data-slot="pill-donut"
    :style="{ '--pill-donut-enter-duration': `${String(ENTER_DURATION_MS)}ms` }"
    :data-entering="entering ? '' : undefined"
    :class="props.class"
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
        :pad-angle="ARC_GAP"
        :corner-radius="arcWidth / 2"
        :duration="UPDATE_DURATION_MS"
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
  block-size: 100%;
  inline-size: 100%;
}

[data-slot='pill-donut-chart'] {
  block-size: 100%;
  inline-size: 100%;
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
