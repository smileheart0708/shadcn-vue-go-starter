<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { UserPlus } from '@lucide/vue'
import AdminUserFormDialog from '@/components/admin-users/AdminUserFormDialog.vue'
import AdminUserStatusConfirmDialog from '@/components/admin-users/AdminUserStatusConfirmDialog.vue'
import AdminUsersFilters from '@/components/admin-users/AdminUsersFilters.vue'
import AdminUsersTable from '@/components/admin-users/AdminUsersTable.vue'
import { Button } from '@/components/ui/button'
import { CAPABILITY } from '@/lib/auth/capabilities.generated'
import { useAuthStore } from '@/stores/auth'
import { useAdminUsersList } from '@/composables/admin-users/useAdminUsersList'
import { useAdminUserMutations } from '@/composables/admin-users/useAdminUserMutations'

const { t } = useI18n()
const authStore = useAuthStore()
const canCreateUsers = computed(() =>
  authStore.can(CAPABILITY.managementUsersCreate),
)

const {
  users,
  total,
  page,
  searchQuery,
  statusFilter,
  loading,
  refreshing,
  loadFailed,
  totalPages,
  loadUsers,
  submitFilters,
  goToPreviousPage,
  goToNextPage,
} = useAdminUsersList()

const {
  dialogOpen,
  dialogMode,
  dialogPending,
  form,
  confirmOpen,
  confirmTarget,
  confirmPending,
  openCreateDialog,
  openEditDialog,
  submitDialog,
  requestToggleStatus,
  confirmToggleStatus,
} = useAdminUserMutations({
  refreshUsers: async () => {
    await loadUsers({ background: true })
  },
})
</script>

<template>
  <div class="flex flex-1 flex-col gap-6 p-4 lg:p-6">
    <section class="flex items-start justify-between gap-4">
      <div class="space-y-1">
        <h2 class="text-2xl font-semibold">{{ t('adminUsers.title') }}</h2>
        <p class="text-sm text-muted-foreground">
          {{ t('adminUsers.description') }}
        </p>
      </div>

      <Button
        :disabled="!canCreateUsers"
        @click="openCreateDialog"
      >
        <UserPlus class="me-2 block-4 inline-4" />
        {{ t('adminUsers.actions.createUser') }}
      </Button>
    </section>

    <AdminUsersFilters
      v-model:search-query="searchQuery"
      v-model:status-filter="statusFilter"
      @submit="submitFilters"
    />

    <AdminUsersTable
      :users="users"
      :loading="loading"
      :refreshing="refreshing"
      :load-failed="loadFailed"
      :page="page"
      :total="total"
      :total-pages="totalPages"
      @retry="loadUsers"
      @previous-page="goToPreviousPage"
      @next-page="goToNextPage"
      @edit-user="openEditDialog"
      @toggle-status="requestToggleStatus"
    />

    <AdminUserFormDialog
      v-model:open="dialogOpen"
      v-model:form="form"
      :mode="dialogMode"
      :pending="dialogPending"
      @submit="submitDialog"
    />

    <AdminUserStatusConfirmDialog
      v-model:open="confirmOpen"
      :target="confirmTarget"
      :pending="confirmPending"
      @confirm="confirmToggleStatus"
    />
  </div>
</template>
