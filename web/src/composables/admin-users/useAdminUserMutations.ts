import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import {
  createAdminUser,
  disableAdminUser,
  enableAdminUser,
  type ManagedUser,
  updateAdminUser,
} from '@/lib/api/admin-users'
import { resolveAPIErrorMessage } from '@/lib/api/error-messages'

interface UseAdminUserMutationsOptions {
  refreshUsers: () => Promise<void>
}

type AdminUserDialogMode = 'create' | 'edit'

export function useAdminUserMutations(options: UseAdminUserMutationsOptions) {
  const { t } = useI18n()

  const dialogOpen = ref(false)
  const dialogMode = ref<AdminUserDialogMode>('create')
  const dialogPending = ref(false)
  const selectedUser = ref<ManagedUser | null>(null)
  const form = ref({
    username: '',
    email: '',
    password: '',
  })

  const confirmOpen = ref(false)
  const confirmTarget = ref<ManagedUser | null>(null)
  const confirmPending = ref(false)

  function openCreateDialog() {
    dialogMode.value = 'create'
    selectedUser.value = null
    form.value = {
      username: '',
      email: '',
      password: '',
    }
    dialogOpen.value = true
  }

  function openEditDialog(user: ManagedUser) {
    dialogMode.value = 'edit'
    selectedUser.value = user
    form.value = {
      username: user.username,
      email: user.email ?? '',
      password: '',
    }
    dialogOpen.value = true
  }

  async function submitDialog() {
    if (dialogPending.value) {
      return
    }

    const creating = dialogMode.value === 'create'
    dialogPending.value = true
    const payload = {
      username: form.value.username.trim(),
      email: form.value.email.trim() === '' ? null : form.value.email.trim(),
    }
    let requestPromise: Promise<ManagedUser>
    if (creating) {
      requestPromise = createAdminUser({
        ...payload,
        password: form.value.password,
      })
    } else {
      const target = selectedUser.value
      if (target === null) {
        dialogPending.value = false
        return
      }
      requestPromise = updateAdminUser(target.id, payload)
    }

    toast.promise(requestPromise, {
      loading: creating
        ? t('adminUsers.feedback.creating')
        : t('adminUsers.feedback.updating'),
      success: () =>
        creating
          ? t('adminUsers.feedback.createSuccess')
          : t('adminUsers.feedback.updateSuccess'),
      error: (error: unknown) =>
        resolveAPIErrorMessage(t, error) ??
        (creating
          ? t('adminUsers.feedback.createFailed')
          : t('adminUsers.feedback.updateFailed')),
    })

    try {
      await requestPromise
      dialogOpen.value = false
      await options.refreshUsers()
    } catch {
      // The rejection is already surfaced by toast.promise above; swallow it here so the
      // shared promise is not left unhandled. Keep the dialog open so the user can retry
      // with corrected input.
    } finally {
      dialogPending.value = false
    }
  }

  function requestToggleStatus(user: ManagedUser) {
    confirmTarget.value = user
    confirmOpen.value = true
  }

  async function confirmToggleStatus() {
    const target = confirmTarget.value
    if (target === null || confirmPending.value) {
      return
    }

    confirmPending.value = true
    const disabling = target.status === 'active'
    const requestPromise = disabling
      ? disableAdminUser(target.id)
      : enableAdminUser(target.id)

    toast.promise(requestPromise, {
      loading: disabling
        ? t('adminUsers.feedback.disabling')
        : t('adminUsers.feedback.enabling'),
      success: () =>
        disabling
          ? t('adminUsers.feedback.disableSuccess')
          : t('adminUsers.feedback.enableSuccess'),
      error: (error: unknown) =>
        resolveAPIErrorMessage(t, error) ??
        (disabling
          ? t('adminUsers.feedback.disableFailed')
          : t('adminUsers.feedback.enableFailed')),
    })

    try {
      await requestPromise
      confirmOpen.value = false
      confirmTarget.value = null
      await options.refreshUsers()
    } catch {
      // Already reported by toast.promise; swallow to avoid an unhandled rejection and
      // leave the confirmation dialog open for a retry.
    } finally {
      confirmPending.value = false
    }
  }

  return {
    dialogOpen,
    dialogMode,
    dialogPending,
    selectedUser,
    form,
    confirmOpen,
    confirmTarget,
    confirmPending,
    openCreateDialog,
    openEditDialog,
    submitDialog,
    requestToggleStatus,
    confirmToggleStatus,
  }
}
