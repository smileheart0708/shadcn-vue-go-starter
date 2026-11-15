import { computed, ref, watch } from 'vue'
import { useCountdown } from '@vueuse/core'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { resolveErrorMessageOrInternal } from '@/lib/api/error-messages'
import { CAPABILITY } from '@/lib/auth/capabilities.generated'
import { useAccountMutations } from '@/composables/settings/useAccountMutations'
import { useAuthStore } from '@/stores/auth'

const DELETE_COUNTDOWN_SECONDS = 3

export function useDeleteAccountFlow() {
  const { t } = useI18n()
  const router = useRouter()
  const authStore = useAuthStore()
  const accountMutations = useAccountMutations()

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
      // Resetting here, not just in closeDeleteAccountFlow(), because the dialog also closes
      // through Cancel, Escape and the overlay — none of which go through that function. The
      // checkbox gates the open button (`:disabled="!canOpenDeleteDialog"`), so leaving it
      // checked let a later click skip the confirmation for a destructive action.
      deleteAccountConfirmed.value = false
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
      await accountMutations.deleteAccount()
      closeDeleteAccountFlow()
      toast.success(t('common.feedback.deleted'))
      await router.push({ name: 'login' })
    } catch (error) {
      toast.error(resolveErrorMessageOrInternal(t, error))
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
