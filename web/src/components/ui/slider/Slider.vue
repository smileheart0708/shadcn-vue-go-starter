<script setup lang="ts">
import type { SliderRootEmits, SliderRootProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { reactiveOmit } from '@vueuse/core'
import {
  SliderRange,
  SliderRoot,
  SliderThumb,
  SliderTrack,
  useForwardPropsEmits,
} from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<
  SliderRootProps & {
    class?: HTMLAttributes['class']
    // reka 只在多 thumb 时自动生成名称（两个是 Minimum/Maximum，更多是 Value N of M），
    // 单 thumb 的可访问名称永远是空的：读屏只会念“滑块 + 数值”，所以调用方必须传这个 prop。
    thumbLabel?: string
    // 可选的气泡，显示的是 SliderRoot 插槽给的实时值，调用方不传值也不重复维护一份状态。
    showThumbValue?: boolean
  }
>()
const emits = defineEmits<SliderRootEmits>()

const delegatedProps = reactiveOmit(
  props,
  'class',
  'thumbLabel',
  'showThumbValue',
)

const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <SliderRoot
    v-slot="{ modelValue: sliderModelValue }"
    data-slot="slider"
    :class="
      cn(
        'group/root relative flex touch-none items-center select-none inline-full data-disabled:opacity-50 data-[orientation=vertical]:flex-col data-[orientation=vertical]:block-full data-[orientation=vertical]:inline-auto data-[orientation=vertical]:min-block-44',
        props.class,
      )
    "
    v-bind="forwarded"
  >
    <SliderTrack
      data-slot="slider-track"
      class="relative grow overflow-hidden rounded-full bg-muted data-[orientation=horizontal]:block-1.5 data-[orientation=horizontal]:inline-full data-[orientation=vertical]:block-full data-[orientation=vertical]:inline-1.5"
    >
      <SliderRange
        data-slot="slider-range"
        class="absolute bg-primary data-[orientation=horizontal]:block-full data-[orientation=vertical]:inline-full"
      />
    </SliderTrack>

    <SliderThumb
      v-for="(value, key) in sliderModelValue"
      :key="key"
      data-slot="slider-thumb"
      :aria-label="props.thumbLabel"
      class="group/thumb block shrink-0 rounded-full border border-primary bg-white shadow-sm ring-ring/50 transition-[color,box-shadow] block-4 inline-4 hover:ring-4 focus-visible:ring-4 focus-visible:outline-hidden disabled:pointer-events-none disabled:opacity-50"
    >
      <!--
        气泡锚在绝对定位的 thumb 上；aria-hidden 是必须的：thumb 是 role="slider"，
        内部文本即使 opacity:0 仍会被读屏当作无障碍值，而数值已由 aria-valuenow 表达。
        shortcut: reka 把 pointer capture 交给 pointerdown 的 target，从轨道起拖时
        :active 只落在 track 及其祖先 root，thumb 恒不 active，所以拖拽显隐必须挂
        group-active/root 而不是 thumb 的 group-active。等 reka 暴露 dragging 状态时
        改用它并移除 group/root。
      -->
      <span
        v-if="props.showThumbValue"
        aria-hidden="true"
        data-slot="slider-thumb-value"
        class="pointer-events-none absolute inset-s-1/2 -inset-bs-8 -translate-x-1/2 rounded-md bg-foreground px-2 py-1 text-xs font-medium whitespace-nowrap text-background tabular-nums opacity-0 shadow-xs transition-opacity group-hover/thumb:opacity-100 group-focus-visible/thumb:opacity-100 group-active/root:opacity-100"
      >
        {{ value }}
      </span>
    </SliderThumb>
  </SliderRoot>
</template>
