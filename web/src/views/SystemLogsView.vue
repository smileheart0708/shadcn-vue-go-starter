<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import PageHeader from '@/components/app/PageHeader.vue'
import SystemLogConsolePanel from '@/components/system-logs/SystemLogConsolePanel.vue'
import { Badge } from '@/components/ui/badge'
import { useSystemLogStream } from '@/composables/system-logs/useSystemLogStream'
import { CAPABILITY } from '@/lib/auth/capabilities.generated'
import { useAuthStore } from '@/stores/auth'
import { useSystemLogsPreferencesStore } from '@/stores/system-logs-preferences'

const { t } = useI18n()
const authStore = useAuthStore()
const systemLogsPreferences = useSystemLogsPreferencesStore()
const { levels, historyLimit, exportFormat } = storeToRefs(
  systemLogsPreferences,
)

const canReadSystemLogs = computed(() =>
  authStore.can(CAPABILITY.managementSystemLogsRead),
)

const {
  entries: systemLogEntries,
  loading: systemLogLoading,
  connecting,
  connected,
  streamError,
  clearEntries,
  reconnect,
} = useSystemLogStream({
  historyLimit,
})
</script>

<template>
  <div
    class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:overflow-hidden lg:p-6 lg:min-block-0"
  >
    <PageHeader :title="t('systemLogs.title')">
      <template #meta>
        <Badge variant="outline">
          {{
            t('systemLogs.summary.buffered', {
              count: systemLogEntries.length,
            })
          }}
        </Badge>
      </template>
    </PageHeader>

    <SystemLogConsolePanel
      v-if="canReadSystemLogs"
      :entries="systemLogEntries"
      :loading="systemLogLoading"
      :connecting="connecting"
      :connected="connected"
      :stream-error="streamError"
      :levels="levels"
      :history-limit="historyLimit"
      :export-format="exportFormat"
      @clear="clearEntries"
      @reconnect="reconnect"
      @update:levels="systemLogsPreferences.setLevels"
      @update:history-limit="systemLogsPreferences.setHistoryLimit"
      @update:export-format="systemLogsPreferences.setExportFormat"
    />
  </div>
</template>
