<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  title: string
  titleClass?: HTMLAttributes['class']
  contentClass?: HTMLAttributes['class']
}>()
</script>

<template>
  <section class="flex flex-col gap-3">
    <!--
      组名只写标题，禁止加描述：组内每行已有自己的说明，组级描述属于重复表述。
      因此这里不提供 description 插槽/属性，需要解释就写进对应设置项的描述里。
    -->
    <h3 :class="cn('text-base font-semibold', props.titleClass)">
      {{ props.title }}
    </h3>

    <!--
      卡片由「组」持有：一处一组，不再给单个设置项套第二层框。
      阴影保持 shadow-xs：与组内带 shadow-xs 的输入类控件（Select/Input/Switch/Checkbox）
      以及 ui/card 卡片同一层级；shadow-sm 的透明度和扩散都明显更重，会让卡片压过控件。
      直接子节点应是 SettingsRow，或一个包裹容器——否则 divide-y 会把分隔线切进表单字段之间。
      包裹容器要自带横向内边距（px-4）与行的内边距对齐；纵向可由内容决定（表单块用 p-4）。
    -->
    <div
      :class="
        cn(
          'divide-y rounded-xl border bg-card text-card-foreground shadow-xs',
          props.contentClass,
        )
      "
    >
      <slot />
    </div>
  </section>
</template>
