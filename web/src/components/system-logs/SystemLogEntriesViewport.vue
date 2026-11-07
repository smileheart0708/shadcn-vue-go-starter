<script setup lang="ts">
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
} from '@/utils/system-logs/console'

defineProps<{
  entries: readonly SystemLogEntry[]
  loading: boolean
  firstLoadPending: boolean
}>()

const { t } = useI18n()
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
    class="space-y-1 p-3 font-mono text-xs/5 whitespace-nowrap min-inline-max"
  >
    <div
      v-for="entry in entries"
      :key="entry.id"
      class="grid grid-cols-[auto_auto_auto_1fr] items-start gap-2 rounded-lg border border-transparent px-3 py-2 transition-colors hover:bg-muted/60"
    >
      <span class="text-muted-foreground tabular-nums">{{
        formatSystemLogTimestamp(entry.timestamp)
      }}</span>
      <Badge :variant="getLevelBadgeVariant(entry.level)">{{
        entry.level
      }}</Badge>
      <span class="text-muted-foreground">{{ entry.source }}</span>
      <span>{{ entry.text }}</span>
    </div>
  </div>
</template>
