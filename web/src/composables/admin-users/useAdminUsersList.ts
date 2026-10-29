import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { resolveAPIErrorMessage } from '@/lib/api/error-messages'
import { listAdminUsers, type ManagedUser } from '@/lib/api/admin-users'
import { getAdminUsersTotalPages } from '@/utils/admin-users/table'

export type AdminUsersStatusFilter = 'ALL' | ManagedUser['status']

export function useAdminUsersList() {
  const { t } = useI18n()

  const users = ref<ManagedUser[]>([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(10)
  const searchQuery = ref('')
  const statusFilter = ref<AdminUsersStatusFilter>('ALL')
  const loading = ref(true)
  const refreshing = ref(false)
  const loadFailed = ref(false)
  const totalPages = computed(() =>
    getAdminUsersTotalPages(total.value, pageSize.value),
  )

  onMounted(() => {
    void loadUsers()
  })

  async function loadUsers(options: { background?: boolean } = {}) {
    if (options.background === true) {
      refreshing.value = true
    } else {
      loading.value = true
    }

    try {
      const response = await listAdminUsers({
        q: searchQuery.value,
        status: statusFilter.value === 'ALL' ? null : statusFilter.value,
        page: page.value,
        pageSize: pageSize.value,
      })

      users.value = response.items
      page.value = response.page
      pageSize.value = response.pageSize
      total.value = response.total
      loadFailed.value = false
    } catch (error) {
      loadFailed.value = true
      toast.error(
        resolveAPIErrorMessage(t, error) ?? t('adminUsers.feedback.loadFailed'),
      )
    } finally {
      loading.value = false
      refreshing.value = false
    }
  }

  function submitFilters() {
    page.value = 1
    void loadUsers()
  }

  function goToPreviousPage() {
    if (page.value <= 1 || refreshing.value) {
      return
    }

    page.value -= 1
    void loadUsers({ background: true })
  }

  function goToNextPage() {
    if (page.value >= totalPages.value || refreshing.value) {
      return
    }

    page.value += 1
    void loadUsers({ background: true })
  }

  return {
    users,
    total,
    page,
    pageSize,
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
  }
}
