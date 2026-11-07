<script setup lang="ts">
import type { SwitchRootEmits, SwitchRootProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { reactiveOmit } from '@vueuse/core'
import { SwitchRoot, SwitchThumb, useForwardPropsEmits } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<
  SwitchRootProps & { class?: HTMLAttributes['class'] }
>()

const emits = defineEmits<SwitchRootEmits>()

const delegatedProps = reactiveOmit(props, 'class')

const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <!-- 同心圆角：轨道半径 = 圆点半径 + 间距；变量对应 block-4 的一半与 px-0.5 + 1px border，改尺寸时同步。 -->
  <SwitchRoot
    v-slot="slotProps"
    data-slot="switch"
    v-bind="forwarded"
    :class="
      cn(
        'peer inline-flex shrink-0 items-center rounded-[calc(var(--switch-thumb-radius)+var(--switch-inset))] border border-transparent px-0.5 shadow-xs transition-all outline-none [--switch-inset:0.1875rem] [--switch-thumb-radius:0.5rem] block-5.5 inline-10 focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=unchecked]:bg-input dark:data-[state=unchecked]:bg-input/80',
        props.class,
      )
    "
  >
    <!-- 圆点固定 16px；位移的 100% 以它自身宽度为基准，行程由轨道尺寸与水平内边距共同决定，改任一侧都要同步。 -->
    <SwitchThumb
      data-slot="switch-thumb"
      :class="
        cn(
          'pointer-events-none block rounded-(--switch-thumb-radius) bg-background ring-0 transition-transform block-4 inline-4 data-[state=checked]:translate-x-[calc(100%+2px)] data-[state=unchecked]:translate-x-0 dark:data-[state=checked]:bg-primary-foreground dark:data-[state=unchecked]:bg-foreground',
        )
      "
    >
      <slot
        name="thumb"
        v-bind="slotProps"
      />
    </SwitchThumb>
  </SwitchRoot>
</template>
