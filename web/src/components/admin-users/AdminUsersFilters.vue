<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { AdminUsersStatusFilter } from '@/composables/admin-users/useAdminUsersList'

const emit = defineEmits<{
  submit: []
}>()

const searchQuery = defineModel<string>('searchQuery', { required: true })
const statusFilter = defineModel<AdminUsersStatusFilter>('statusFilter', {
  required: true,
})

const { t } = useI18n()
</script>

<template>
  <section class="flex gap-3">
    <Input
      v-model="searchQuery"
      :placeholder="t('adminUsers.filters.searchPlaceholder')"
      class="flex-1 min-inline-0"
      @keydown.enter="emit('submit')"
    />

    <Select
      v-model="statusFilter"
      @update:model-value="emit('submit')"
    >
      <SelectTrigger class="shrink-0 inline-auto">
        <SelectValue :placeholder="t('adminUsers.filters.statusPlaceholder')" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="ALL">{{
          t('adminUsers.filters.statusAll')
        }}</SelectItem>
        <SelectItem value="active">{{ t('common.status.active') }}</SelectItem>
        <SelectItem value="disabled">{{
          t('common.status.disabled')
        }}</SelectItem>
      </SelectContent>
    </Select>
  </section>
</template>
