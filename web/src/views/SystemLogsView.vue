<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AppShellPage from '@/components/app-shell/AppShellPage.vue'
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
  <AppShellPage variant="fill">
    <template #meta>
      <Badge variant="outline">
        {{
          t('systemLogs.summary.buffered', {
            count: systemLogEntries.length,
          })
        }}
      </Badge>
    </template>

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
  </AppShellPage>
</template>
