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
  }
>()
const emits = defineEmits<SliderRootEmits>()

const delegatedProps = reactiveOmit(props, 'class', 'thumbLabel')

const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <SliderRoot
    v-slot="{ modelValue: sliderModelValue }"
    data-slot="slider"
    :class="
      cn(
        'relative flex touch-none items-center select-none inline-full data-disabled:opacity-50 data-[orientation=vertical]:flex-col data-[orientation=vertical]:block-full data-[orientation=vertical]:inline-auto data-[orientation=vertical]:min-block-44',
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
      v-for="(_, key) in sliderModelValue"
      :key="key"
      data-slot="slider-thumb"
      :aria-label="props.thumbLabel"
      class="block shrink-0 rounded-full border border-primary bg-white shadow-sm ring-ring/50 transition-[color,box-shadow] block-4 inline-4 hover:ring-4 focus-visible:ring-4 focus-visible:outline-hidden disabled:pointer-events-none disabled:opacity-50"
    />
  </SliderRoot>
</template>
