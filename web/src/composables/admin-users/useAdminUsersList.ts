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
  // Sent on every request, so this value (not the server fallback) defines the page size.
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

  // Every request takes a ticket and only the newest one may write state. Paging and
  // filtering can overlap — the status select submits immediately while a page request is
  // still in flight — and an older response used to write its own `page` back, leaving
  // the table on page 2 while the filter showed the new query.
  let requestTicket = 0

  async function loadUsers(
    options: { background?: boolean; page?: number } = {},
  ) {
    const ticket = ++requestTicket
    // The target page is a parameter rather than a mutation of `page` before the request: a
    // failed request used to leave `page` already advanced while the table still showed the
    // previous rows, and `loadFailed` hides the pagination bar, so there was no way back to the
    // page the user came from except reloading the whole document.
    const requestedPage = options.page ?? page.value
    if (options.background === true) {
      refreshing.value = true
    } else {
      loading.value = true
    }

    try {
      const response = await listAdminUsers({
        q: searchQuery.value,
        status: statusFilter.value === 'ALL' ? null : statusFilter.value,
        page: requestedPage,
        pageSize: pageSize.value,
      })

      if (ticket !== requestTicket) {
        return
      }

      users.value = response.items
      page.value = response.page
      pageSize.value = response.pageSize
      total.value = response.total
      loadFailed.value = false
    } catch (error) {
      if (ticket !== requestTicket) {
        return
      }

      loadFailed.value = true
      toast.error(
        resolveAPIErrorMessage(t, error) ?? t('common.feedback.loadFailed'),
      )
    } finally {
      // A superseded request must not clear the flags the newer request owns, or the
      // skeleton flashes off before its own data arrives.
      if (ticket === requestTicket) {
        loading.value = false
        refreshing.value = false
      }
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

    void loadUsers({ background: true, page: page.value - 1 })
  }

  function goToNextPage() {
    if (page.value >= totalPages.value || refreshing.value) {
      return
    }

    void loadUsers({ background: true, page: page.value + 1 })
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
