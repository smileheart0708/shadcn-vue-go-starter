<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowDownToLine,
  Download,
  Pause,
  RefreshCw,
  Trash2,
} from '@lucide/vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  SYSTEM_LOG_HISTORY_LIMIT_VALUES,
  type SystemLogEntry,
  type SystemLogHistoryLimit,
  type SystemLogLevel,
} from '@/lib/api/system-logs'
import { cn } from '@/lib/utils'
import type { SystemLogExportFormat } from '@/stores/system-logs-preferences'
import { useSystemLogViewport } from '@/composables/system-logs/useSystemLogViewport'
import {
  getConnectionIndicatorClass,
  normalizeHistoryLimitSelectValue,
  selectVisibleSystemLogEntries,
} from '@/utils/system-logs/console'
import SystemLogEntriesViewport from './SystemLogEntriesViewport.vue'
import SystemLogLevelMultiSelect from './SystemLogLevelMultiSelect.vue'
import SystemLogsExportDialog from './SystemLogsExportDialog.vue'

const props = defineProps<{
  entries: readonly SystemLogEntry[]
  loading: boolean
  connecting: boolean
  connected: boolean
  streamError: string
  levels: readonly SystemLogLevel[]
  historyLimit: SystemLogHistoryLimit
  exportFormat: SystemLogExportFormat
}>()

const emit = defineEmits<{
  clear: []
  reconnect: []
  'update:levels': [value: SystemLogLevel[]]
  'update:history-limit': [value: SystemLogHistoryLimit]
  'update:export-format': [value: SystemLogExportFormat]
}>()

const { t } = useI18n()

const searchQuery = ref('')
const exportDialogOpen = ref(false)
const { autoScroll, toggleAutoScroll, scrollToBottom, handleViewportScroll } =
  useSystemLogViewport()

const visibleEntries = computed(() =>
  selectVisibleSystemLogEntries({
    entries: props.entries,
    historyLimit: props.historyLimit,
    levels: props.levels,
    searchQuery: searchQuery.value,
  }),
)
const connectionStatusLabel = computed(() => {
  if (props.connecting) {
    return t('systemLogs.connection.connecting')
  }
  return props.connected
    ? t('systemLogs.connection.connected')
    : t('systemLogs.connection.disconnected')
})

// 穷尽 switch：i18n lint 只认 t() 调用里的字面量键。
function historyLimitLabel(limit: SystemLogHistoryLimit): string {
  switch (limit) {
    case 'ALL':
      return t('systemLogs.filters.history.ALL')
    case 100:
      return t('systemLogs.filters.history.100')
    case 200:
      return t('systemLogs.filters.history.200')
    case 500:
      return t('systemLogs.filters.history.500')
    case 1000:
      return t('systemLogs.filters.history.1000')
  }
}
const connectionIndicatorClass = computed(() =>
  getConnectionIndicatorClass(props.connecting, props.connected),
)
const exportFormatModel = computed({
  get: () => props.exportFormat,
  set: (value: SystemLogExportFormat) => {
    emit('update:export-format', value)
  },
})

watch(
  () => props.entries.length,
  () => {
    if (autoScroll.value) {
      void scrollToBottom()
    }
  },
  { flush: 'post' },
)

function handleHistoryLimitChange(value: unknown) {
  const nextLimit = normalizeHistoryLimitSelectValue(value)
  if (nextLimit === null) {
    return
  }
  emit('update:history-limit', nextLimit)
}
</script>

<template>
  <Card class="lg:flex-1 lg:overflow-hidden lg:min-block-0">
    <CardHeader
      class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
    >
      <div class="flex flex-wrap items-center gap-2 min-inline-0">
        <CardTitle class="text-lg">{{
          t('systemLogs.console.title')
        }}</CardTitle>
        <span
          class="relative inline-flex items-center justify-center block-3 inline-3"
          :aria-label="connectionStatusLabel"
          role="status"
        >
          <span
            v-if="connected"
            class="absolute inline-flex animate-ping rounded-full bg-emerald-500 opacity-60 block-full inline-full"
          />
          <span
            :class="
              cn(
                'relative inline-flex rounded-full shadow-sm ring-2 ring-background block-3 inline-3',
                connectionIndicatorClass,
              )
            "
          />
        </span>
        <Badge
          v-if="streamError"
          variant="destructive"
          class="truncate max-inline-full"
        >
          {{ streamError }}
        </Badge>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          :disabled="entries.length === 0"
          @click="exportDialogOpen = true"
        >
          <Download class="block-4 inline-4" />
          {{ t('systemLogs.actions.export') }}
        </Button>
        <Button
          variant="outline"
          size="sm"
          @click="emit('clear')"
        >
          <Trash2 class="block-4 inline-4" />
          {{ t('systemLogs.actions.clear') }}
        </Button>
        <Button
          variant="outline"
          size="sm"
          @click="toggleAutoScroll"
        >
          <Pause
            v-if="autoScroll"
            class="block-4 inline-4"
          />
          <ArrowDownToLine
            v-else
            class="block-4 inline-4"
          />
          {{
            autoScroll
              ? t('systemLogs.actions.pauseFollow')
              : t('systemLogs.actions.resumeFollow')
          }}
        </Button>
        <Button
          variant="outline"
          size="sm"
          :disabled="connecting"
          @click="emit('reconnect')"
        >
          <RefreshCw
            :class="cn('block-4 inline-4', connecting && 'animate-spin')"
          />
          {{ t('systemLogs.actions.reconnect') }}
        </Button>
      </div>
    </CardHeader>

    <CardContent class="flex flex-col gap-4 lg:flex-1 lg:min-block-0">
      <div class="flex flex-col gap-2 md:flex-row md:items-center">
        <Input
          v-model="searchQuery"
          :placeholder="t('systemLogs.filters.searchPlaceholder')"
          class="md:flex-1"
        />

        <SystemLogLevelMultiSelect
          :model-value="levels"
          @update:model-value="emit('update:levels', $event)"
        />

        <Select
          :model-value="historyLimit"
          @update:model-value="handleHistoryLimitChange"
        >
          <SelectTrigger class="inline-full md:inline-auto">
            <SelectValue
              :placeholder="t('systemLogs.filters.historyPlaceholder')"
            />
          </SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="limit in SYSTEM_LOG_HISTORY_LIMIT_VALUES"
              :key="String(limit)"
              :value="limit"
            >
              {{ historyLimitLabel(limit) }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div
        class="flex flex-1 flex-col rounded-xl border bg-sidebar/40 min-block-0"
      >
        <div
          ref="viewport"
          class="overflow-auto rounded-xl block-128 lg:flex-1 lg:block-auto lg:min-block-0"
          @scroll="handleViewportScroll"
        >
          <SystemLogEntriesViewport
            :entries="visibleEntries"
            :loading="loading"
          />
        </div>
      </div>
    </CardContent>

    <SystemLogsExportDialog
      v-model:open="exportDialogOpen"
      v-model:format="exportFormatModel"
      :entries="entries"
      :history-limit="historyLimit"
      :levels="levels"
    />
  </Card>
</template>
