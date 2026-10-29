<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  SYSTEM_LOG_LEVEL_VALUES,
  type SystemLogLevel,
} from '@/lib/api/system-logs'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    modelValue?: readonly SystemLogLevel[]
    class?: HTMLAttributes['class']
  }>(),
  {
    modelValue: () => [...SYSTEM_LOG_LEVEL_VALUES],
  },
)

const emit = defineEmits<{
  'update:model-value': [value: SystemLogLevel[]]
}>()

const { t } = useI18n()

// 穷尽 switch：i18n lint 只认 t() 调用里的字面量键，模板字符串键会被判为未使用。
function levelLabel(level: SystemLogLevel): string {
  switch (level) {
    case 'DEBUG':
      return t('systemLogs.filters.level.DEBUG')
    case 'INFO':
      return t('systemLogs.filters.level.INFO')
    case 'WARN':
      return t('systemLogs.filters.level.WARN')
    case 'ERROR':
      return t('systemLogs.filters.level.ERROR')
  }
}

const selectedLevels = computed(() => new Set(props.modelValue))
const orderedSelectedLevels = computed(() =>
  SYSTEM_LOG_LEVEL_VALUES.filter((level) => selectedLevels.value.has(level)),
)
const triggerLabel = computed(() => {
  if (orderedSelectedLevels.value.length === SYSTEM_LOG_LEVEL_VALUES.length) {
    return t('systemLogs.filters.level.allSelected')
  }
  if (orderedSelectedLevels.value.length === 0) {
    return t('systemLogs.filters.level.noneSelected')
  }
  return orderedSelectedLevels.value.map(levelLabel).join(', ')
})

function updateLevel(level: SystemLogLevel, selected: boolean) {
  const nextLevels = new Set(props.modelValue)
  if (selected) {
    nextLevels.add(level)
  } else {
    nextLevels.delete(level)
  }

  emit(
    'update:model-value',
    SYSTEM_LOG_LEVEL_VALUES.filter((candidate) => nextLevels.has(candidate)),
  )
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        type="button"
        variant="outline"
        :class="
          cn('justify-between gap-2 inline-full md:inline-auto', props.class)
        "
      >
        <span class="truncate">{{ triggerLabel }}</span>
        <ChevronDown class="shrink-0 opacity-60 block-4 inline-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent
      align="end"
      class="inline-auto"
    >
      <DropdownMenuCheckboxItem
        v-for="level in SYSTEM_LOG_LEVEL_VALUES"
        :key="level"
        :model-value="selectedLevels.has(level)"
        @select.prevent
        @update:model-value="updateLevel(level, $event === true)"
      >
        {{ levelLabel(level) }}
      </DropdownMenuCheckboxItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
