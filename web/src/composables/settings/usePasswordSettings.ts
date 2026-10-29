import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { resolveAPIErrorMessage } from '@/lib/api/error-messages'
import { useAccountStore } from '@/stores/account'

export function usePasswordSettings() {
  const { t } = useI18n()
  const router = useRouter()
  const accountStore = useAccountStore()

  const passwordError = ref('')
  const passwordForm = ref({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  })
  const isUpdatingPassword = ref(false)

  async function updatePassword() {
    passwordError.value = ''

    if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
      passwordError.value = t('settings.account.passwordMismatch')
      return
    }

    isUpdatingPassword.value = true

    try {
      await accountStore.changePassword({
        currentPassword: passwordForm.value.currentPassword,
        newPassword: passwordForm.value.newPassword,
      })

      passwordForm.value = {
        currentPassword: '',
        newPassword: '',
        confirmPassword: '',
      }

      toast.success(t('settings.account.passwordUpdated'))
      await router.push({ name: 'login' })
    } catch (error) {
      const message =
        resolveAPIErrorMessage(t, error) ?? t('apiError.internal_error')
      passwordError.value = message
      toast.error(message)
    } finally {
      isUpdatingPassword.value = false
    }
  }

  return {
    passwordForm,
    passwordError,
    isUpdatingPassword,
    updatePassword,
  }
}
