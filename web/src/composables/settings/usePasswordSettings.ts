import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { resolveErrorMessageOrInternal } from '@/lib/api/error-messages'
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
      passwordError.value = t('common.validation.passwordMismatch')
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

      toast.success(t('common.feedback.updated'))
      await router.push({ name: 'login' })
    } catch (error) {
      // `passwordError` is rendered under the form, so a toast here said the same thing
      // twice and could drift to a second wording.
      passwordError.value = resolveErrorMessageOrInternal(t, error)
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
