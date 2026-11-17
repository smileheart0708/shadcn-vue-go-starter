<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { SystemLogEntry } from '@/lib/api/system-logs'
import {
  SYSTEM_LOG_EXPORT_FORMAT_VALUES,
  type SystemLogExportFormat,
} from '@/stores/system-logs-preferences'
import {
  createSystemLogExportBlob,
  getSystemLogExportFileName,
} from '@/utils/system-logs/export'

interface Props {
  // 名字带 visible 是有意的：这里只收列表已经算好的那批条目（级别/搜索/historyLimit
  // 都由 selectVisibleSystemLogEntries 处理）。对话框自己再过滤一次，就会重演
  // "屏幕 3 条、导出全量" —— 两份过滤实现迟早漂移。
  visibleEntries: readonly SystemLogEntry[]
  open: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const { t, locale } = useI18n()

// 穷尽 switch：i18n lint 只认 t() 调用里的字面量键。
function exportFormatLabel(format: SystemLogExportFormat): string {
  switch (format) {
    case 'csv':
      return t('systemLogs.export.formats.csv')
    case 'txt':
      return t('systemLogs.export.formats.txt')
    case 'json':
      return t('systemLogs.export.formats.json')
  }
}

const exportFormat = defineModel<SystemLogExportFormat>('format', {
  default: 'csv',
})

const openModel = computed({
  get: () => props.open,
  set: (value: boolean) => {
    emit('update:open', value)
  },
})

function handleExport() {
  const entries = props.visibleEntries
  if (entries.length === 0) {
    toast.error(t('systemLogs.feedback.exportEmpty'))
    return
  }

  downloadSystemLogs(entries, exportFormat.value)

  toast.success(
    t('systemLogs.feedback.exportSuccess', { count: entries.length }),
  )
  openModel.value = false
}

function downloadSystemLogs(
  entries: readonly SystemLogEntry[],
  format: SystemLogExportFormat,
) {
  const blob = createSystemLogExportBlob(entries, format, locale.value)
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')

  link.href = url
  link.download = getSystemLogExportFileName(format)
  link.click()
  window.URL.revokeObjectURL(url)
}
</script>

<template>
  <Dialog v-model:open="openModel">
    <DialogContent class="sm:max-inline-md">
      <DialogHeader>
        <DialogTitle>{{ t('systemLogs.export.title') }}</DialogTitle>
        <DialogDescription>
          {{
            t('systemLogs.export.description', { count: visibleEntries.length })
          }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <div class="space-y-2">
          <Label>{{ t('systemLogs.export.fields.format') }}</Label>
          <Select v-model="exportFormat">
            <SelectTrigger>
              <SelectValue
                :placeholder="t('systemLogs.export.fields.format')"
              />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="format in SYSTEM_LOG_EXPORT_FORMAT_VALUES"
                :key="format"
                :value="format"
              >
                {{ exportFormatLabel(format) }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <p class="text-sm text-muted-foreground">
          {{ t('systemLogs.export.preview', { count: visibleEntries.length }) }}
        </p>
      </div>

      <DialogFooter>
        <Button
          variant="outline"
          @click="openModel = false"
        >
          {{ t('common.action.cancel') }}
        </Button>
        <Button @click="handleExport">
          {{ t('systemLogs.actions.export') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
