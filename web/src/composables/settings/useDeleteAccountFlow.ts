import { computed, ref, watch } from 'vue'
import { useCountdown } from '@vueuse/core'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { CAPABILITY } from '@/lib/auth/roles'
import { useAccountStore } from '@/stores/account'
import { useAuthStore } from '@/stores/auth'

const DELETE_COUNTDOWN_SECONDS = 3

export function useDeleteAccountFlow() {
  const { t } = useI18n()
  const router = useRouter()
  const authStore = useAuthStore()
  const accountStore = useAccountStore()

  const deleteDialogOpen = ref(false)
  const deleteAccountConfirmed = ref(false)
  const isDeletingAccount = ref(false)
  const countdown = useCountdown(DELETE_COUNTDOWN_SECONDS)

  const canDeleteAccount = computed(() =>
    authStore.can(CAPABILITY.accountDeleteSelf),
  )
  const deleteCountdown = computed(() =>
    deleteDialogOpen.value ? countdown.remaining.value : 0,
  )
  const canOpenDeleteDialog = computed(
    () =>
      canDeleteAccount.value &&
      deleteAccountConfirmed.value &&
      !isDeletingAccount.value,
  )
  const canSubmitDelete = computed(
    () => canOpenDeleteDialog.value && deleteCountdown.value <= 0,
  )

  watch(canDeleteAccount, (allowed) => {
    if (allowed) {
      return
    }

    closeDeleteAccountFlow()
  })

  watch(deleteDialogOpen, (isOpen) => {
    if (!isOpen) {
      countdown.stop()
      return
    }

    countdown.start(DELETE_COUNTDOWN_SECONDS)
  })

  function closeDeleteAccountFlow() {
    deleteDialogOpen.value = false
    deleteAccountConfirmed.value = false
    countdown.stop()
  }

  async function confirmDelete() {
    if (!canSubmitDelete.value || isDeletingAccount.value) {
      return
    }

    isDeletingAccount.value = true

    try {
      await accountStore.deleteAccount()
      closeDeleteAccountFlow()
      toast.success(t('settings.account.deleteAccountSuccess'))
      await router.push({ name: 'login' })
    } catch (error) {
      toast.error(getAPIErrorMessage(t, error, 'apiError.accountDeleteFailed'))
    } finally {
      isDeletingAccount.value = false
    }
  }

  return {
    deleteDialogOpen,
    deleteCountdown,
    deleteAccountConfirmed,
    canDeleteAccount,
    canOpenDeleteDialog,
    canSubmitDelete,
    isDeletingAccount,
    confirmDelete,
  }
}
