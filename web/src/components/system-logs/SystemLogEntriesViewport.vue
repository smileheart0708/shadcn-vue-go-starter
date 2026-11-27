<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Badge } from '@/components/ui/badge'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import type { SystemLogEntry } from '@/lib/api/system-logs'
import {
  formatSystemLogTimestamp,
  getLevelBadgeVariant,
  selectSystemLogEntryWindow,
  SYSTEM_LOG_ENTRY_ROW_PITCH,
} from '@/utils/system-logs/console'

const props = defineProps<{
  entries: readonly SystemLogEntry[]
  scrollTop: number
  viewportHeight: number
  loading: boolean
  firstLoadPending: boolean
}>()

const { t, locale } = useI18n()

const entryWindow = computed(() =>
  selectSystemLogEntryWindow({
    entryCount: props.entries.length,
    scrollTop: props.scrollTop,
    viewportHeight: props.viewportHeight,
  }),
)
const windowEntries = computed(() =>
  props.entries.slice(entryWindow.value.start, entryWindow.value.end),
)
const leadingSpacerHeight = computed(
  () => entryWindow.value.start * SYSTEM_LOG_ENTRY_ROW_PITCH,
)
const trailingSpacerHeight = computed(
  () =>
    (props.entries.length - entryWindow.value.end) * SYSTEM_LOG_ENTRY_ROW_PITCH,
)
</script>

<template>
  <div
    v-if="loading"
    class="space-y-2 p-3"
  >
    <Skeleton
      v-for="index in 8"
      :key="index"
      class="rounded-lg block-11"
    />
  </div>

  <!--
    首屏结果还没到，而骨架被 useStableLoading 的抑制窗口压住：此时既不是「空」，
    也不该画空态（会闪一下「暂无日志」），留白即可。
  -->
  <template v-else-if="firstLoadPending"></template>

  <div
    v-else-if="entries.length === 0"
    class="p-10"
  >
    <Empty>
      <EmptyHeader>
        <EmptyTitle>{{ t('systemLogs.empty.title') }}</EmptyTitle>
        <EmptyDescription>{{
          t('systemLogs.empty.description')
        }}</EmptyDescription>
      </EmptyHeader>
      <EmptyContent />
    </Empty>
  </div>

  <div
    v-else
    class="p-3 font-mono text-xs/5 whitespace-nowrap min-inline-max"
  >
    <!-- Spacers keep the scrollbar sized for the whole filtered buffer while only the
         rows inside the viewport window are mounted. -->
    <div :style="{ height: `${leadingSpacerHeight}px` }" />

    <div
      v-for="entry in windowEntries"
      :key="entry.id"
      class="mbe-1 grid grid-cols-[auto_auto_auto_1fr] items-start gap-2 rounded-lg border border-transparent px-3 py-2 transition-colors block-10 hover:bg-muted/60"
    >
      <span class="text-muted-foreground tabular-nums">{{
        formatSystemLogTimestamp(entry.timestamp, locale)
      }}</span>
      <Badge :variant="getLevelBadgeVariant(entry.level)">{{
        entry.level
      }}</Badge>
      <span class="text-muted-foreground">{{ entry.source }}</span>
      <span>{{ entry.text }}</span>
    </div>

    <div :style="{ height: `${trailingSpacerHeight}px` }" />
  </div>
</template>
