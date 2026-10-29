<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ShieldBan, ShieldCheck } from '@lucide/vue'
import UserAvatar from '@/components/common/UserAvatar.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { ManagedUser } from '@/lib/api/admin-users'
import { getUserRoleBadgeVariant, getUserRoleLabel } from '@/lib/auth/roles'
import {
  createAdminUserDateFormatter,
  formatNullableAdminUserDateTime,
  getAdminUserStatusBadgeVariant,
  hasAdminUserAction,
} from '@/utils/admin-users/table'

defineProps<{
  users: readonly ManagedUser[]
  loading: boolean
  refreshing: boolean
  loadFailed: boolean
  page: number
  total: number
  totalPages: number
}>()

const emit = defineEmits<{
  retry: []
  'previous-page': []
  'next-page': []
  'edit-user': [user: ManagedUser]
  'toggle-status': [user: ManagedUser]
}>()

const { t, locale } = useI18n()

const dateTimeFormatter = computed(() =>
  createAdminUserDateFormatter(locale.value),
)
const neverUsedLabel = computed(() => t('common.state.neverUsed'))
</script>

<template>
  <div
    v-if="loading"
    class="overflow-hidden rounded-lg border"
  >
    <Table>
      <TableHeader class="bg-muted">
        <TableRow class="hover:bg-transparent">
          <TableHead>{{ t('adminUsers.table.user') }}</TableHead>
          <TableHead>{{ t('adminUsers.table.email') }}</TableHead>
          <TableHead>{{ t('adminUsers.table.role') }}</TableHead>
          <TableHead>{{ t('adminUsers.table.status') }}</TableHead>
          <TableHead>{{ t('adminUsers.table.lastActiveAt') }}</TableHead>
          <TableHead>{{ t('adminUsers.table.createdAt') }}</TableHead>
          <TableHead class="text-end">{{
            t('adminUsers.table.actions')
          }}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow
          v-for="index in 8"
          :key="index"
        >
          <TableCell
            v-for="cell in 7"
            :key="cell"
          >
            <Skeleton class="rounded-md block-5" />
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>

  <div
    v-else-if="loadFailed"
    class="rounded-xl border bg-card p-4"
  >
    <Empty>
      <EmptyHeader>
        <EmptyTitle>{{ t('adminUsers.feedback.loadFailedTitle') }}</EmptyTitle>
        <EmptyDescription>{{
          t('adminUsers.feedback.loadFailed')
        }}</EmptyDescription>
      </EmptyHeader>
      <EmptyContent>
        <Button @click="emit('retry')">{{
          t('adminUsers.actions.retry')
        }}</Button>
      </EmptyContent>
    </Empty>
  </div>

  <div
    v-else
    class="space-y-4"
  >
    <div class="overflow-hidden rounded-lg border">
      <Table>
        <TableHeader class="bg-muted">
          <TableRow class="hover:bg-transparent">
            <TableHead>{{ t('adminUsers.table.user') }}</TableHead>
            <TableHead>{{ t('adminUsers.table.email') }}</TableHead>
            <TableHead>{{ t('adminUsers.table.role') }}</TableHead>
            <TableHead>{{ t('adminUsers.table.status') }}</TableHead>
            <TableHead>{{ t('adminUsers.table.lastActiveAt') }}</TableHead>
            <TableHead>{{ t('adminUsers.table.createdAt') }}</TableHead>
            <TableHead class="text-end">{{
              t('adminUsers.table.actions')
            }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="users.length > 0">
            <TableRow
              v-for="user in users"
              :key="user.id"
            >
              <TableCell>
                <div class="flex items-center gap-3 min-inline-0">
                  <UserAvatar
                    :username="user.username"
                    :avatar-url="user.avatarUrl"
                    class="rounded-lg block-8 inline-8"
                    fallback-class="rounded-lg"
                  />
                  <span class="truncate font-medium">{{ user.username }}</span>
                </div>
              </TableCell>
              <TableCell>{{
                user.email ?? t('adminUsers.table.noEmail')
              }}</TableCell>
              <TableCell>
                <Badge :variant="getUserRoleBadgeVariant(user.role)">
                  {{ getUserRoleLabel(t, user.role) }}
                </Badge>
              </TableCell>
              <TableCell>
                <Badge :variant="getAdminUserStatusBadgeVariant(user.status)">
                  {{
                    user.status === 'active'
                      ? t('adminUsers.status.active')
                      : t('adminUsers.status.disabled')
                  }}
                </Badge>
              </TableCell>
              <TableCell>{{
                formatNullableAdminUserDateTime(
                  user.lastActiveAt,
                  dateTimeFormatter,
                  neverUsedLabel,
                )
              }}</TableCell>
              <TableCell>{{
                formatNullableAdminUserDateTime(
                  user.createdAt,
                  dateTimeFormatter,
                  neverUsedLabel,
                )
              }}</TableCell>
              <TableCell>
                <div
                  v-if="user.role === 'owner'"
                  class="text-end text-sm text-muted-foreground"
                >
                  {{ t('adminUsers.table.ownerReadonly') }}
                </div>
                <div
                  v-else
                  class="flex justify-end gap-2"
                >
                  <Button
                    variant="outline"
                    size="sm"
                    :disabled="!hasAdminUserAction(user, 'update')"
                    @click="emit('edit-user', user)"
                  >
                    {{ t('common.action.edit') }}
                  </Button>
                  <Button
                    size="sm"
                    :variant="
                      user.status === 'active' ? 'destructive' : 'outline'
                    "
                    :disabled="
                      !(
                        hasAdminUserAction(user, 'disable') ||
                        hasAdminUserAction(user, 'enable')
                      )
                    "
                    @click="emit('toggle-status', user)"
                  >
                    <component
                      :is="user.status === 'active' ? ShieldBan : ShieldCheck"
                      class="me-2 block-4 inline-4"
                    />
                    {{
                      user.status === 'active'
                        ? t('adminUsers.actions.disable')
                        : t('adminUsers.actions.enable')
                    }}
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          </template>

          <TableEmpty
            v-else
            :colspan="7"
          >
            {{ t('adminUsers.table.empty') }}
          </TableEmpty>
        </TableBody>
      </Table>
    </div>

    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2 text-sm text-muted-foreground">
        <Spinner
          v-if="refreshing"
          class="block-4 inline-4"
        />
        {{ t('adminUsers.table.pageSummary', { page, totalPages, total }) }}
      </div>

      <div class="flex gap-2">
        <Button
          size="sm"
          variant="outline"
          :disabled="page <= 1 || refreshing"
          @click="emit('previous-page')"
        >
          {{ t('adminUsers.actions.previousPage') }}
        </Button>
        <Button
          size="sm"
          variant="outline"
          :disabled="page >= totalPages || refreshing"
          @click="emit('next-page')"
        >
          {{ t('adminUsers.actions.nextPage') }}
        </Button>
      </div>
    </div>
  </div>
</template>
