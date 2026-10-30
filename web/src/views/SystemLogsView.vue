<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed } from 'vue'
import SystemLogConsolePanel from '@/components/system-logs/SystemLogConsolePanel.vue'
import SystemLogsPageHeader from '@/components/system-logs/SystemLogsPageHeader.vue'
import { useSystemLogStream } from '@/composables/system-logs/useSystemLogStream'
import { CAPABILITY } from '@/lib/auth/capabilities.generated'
import { useAuthStore } from '@/stores/auth'
import { useSystemLogsPreferencesStore } from '@/stores/system-logs-preferences'

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
    <SystemLogsPageHeader :buffered-count="systemLogEntries.length" />

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
