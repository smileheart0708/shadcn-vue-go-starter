<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ShieldBan, ShieldCheck } from '@lucide/vue'
import UserAvatar from '@/components/common/UserAvatar.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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

const props = defineProps<{
  users: readonly ManagedUser[]
  loading: boolean
  refreshing: boolean
  loadFailed: boolean
  page: number
  pageSize: number
  total: number
  totalPages: number
}>()

const emit = defineEmits<{
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

const showSkeleton = computed(() => props.loading || props.loadFailed)

// 失败后骨架停在原位并停止脉冲：继续闪动等于谎称"仍在加载"，而此刻并没有在途请求。
// 但背景刷新（创建/启停用户后的静默重载）必须重新给出反馈，否则用户会看到
// "操作成功"的 toast 配一张静止的灰表，以为操作没生效。
const isFailed = computed(
  () => props.loadFailed && !props.loading && !props.refreshing,
)
const skeletonClass = computed(() => ({ 'animate-none': isFailed.value }))
</script>

<template>
  <!--
    骨架与真实行共用同一张表和同一个表头：列定义只写一次，加列/改列时同步两个 tbody 即可，
    不会再出现"骨架表头和真实表头各写一份"的漂移。骨架行数取当前页容量，行高由首列的
    32px 头像占位决定（与真实行一致），数据到达后在同一布局里就地填空，页头不重绘。
    失败时骨架留在原位并停止脉冲、整体降对比：它表示"未就绪"，不表示"正在加载"。
    原因由 useAdminUsersList 的 toast 承担，页面内不再重复一次；骨架是纯视觉占位，
    所以失败时对辅助技术隐藏，避免被读成一张"没有数据"的空表。
  -->
  <div class="space-y-4">
    <div
      class="overflow-hidden rounded-lg border"
      :class="{ 'opacity-60': isFailed }"
      :aria-busy="loading || refreshing"
      :aria-hidden="isFailed"
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

        <TableBody v-if="showSkeleton">
          <TableRow
            v-for="index in pageSize"
            :key="index"
          >
            <TableCell>
              <div class="flex items-center gap-3">
                <Skeleton
                  class="rounded-lg block-8 inline-8"
                  :class="skeletonClass"
                />
                <Skeleton
                  class="rounded-md block-4 inline-20"
                  :class="skeletonClass"
                />
              </div>
            </TableCell>
            <TableCell>
              <Skeleton
                class="rounded-md block-4 inline-48"
                :class="skeletonClass"
              />
            </TableCell>
            <TableCell>
              <Skeleton
                class="rounded-md block-4 inline-16"
                :class="skeletonClass"
              />
            </TableCell>
            <TableCell>
              <Skeleton
                class="rounded-md block-4 inline-12"
                :class="skeletonClass"
              />
            </TableCell>
            <TableCell>
              <Skeleton
                class="rounded-md block-4 inline-32"
                :class="skeletonClass"
              />
            </TableCell>
            <TableCell>
              <Skeleton
                class="rounded-md block-4 inline-32"
                :class="skeletonClass"
              />
            </TableCell>
            <TableCell>
              <div class="flex justify-end gap-2">
                <Skeleton
                  class="rounded-md block-4 inline-20"
                  :class="skeletonClass"
                />
                <Skeleton
                  class="rounded-md block-4 inline-24"
                  :class="skeletonClass"
                />
              </div>
            </TableCell>
          </TableRow>
        </TableBody>

        <TableBody v-else>
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
                <!-- Read-only is decided by the server's action list, not by role:
                     it already encodes "cannot act on self / peers / higher rank". -->
                <div
                  v-if="user.actions.length === 0"
                  class="text-end text-sm text-muted-foreground"
                >
                  {{ t('adminUsers.table.readonly') }}
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

    <!-- 骨架阶段不渲染分页条：页码在数据回来前没有意义，渲染出来只会上下跳一次。 -->
    <div
      v-if="!showSkeleton"
      class="flex items-center justify-between"
    >
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
