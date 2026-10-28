import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import {
  getAdminSystemSettings,
  updateSystemSettings,
  type SystemSettings,
} from '@/lib/api/system-settings'
import { useAuthStore } from '@/stores/auth'

export function useSystemSettings() {
  const { t } = useI18n()
  const authStore = useAuthStore()

  const loading = ref(true)
  const saving = ref(false)
  const loadFailed = ref(false)
  // `settings` holds the last server-confirmed state; `form` is the editable draft.
  // Keeping both is what makes `isDirty` (and therefore the save/cancel buttons) work.
  const settings = ref<SystemSettings | null>(null)
  const form = ref<SystemSettings>({
    publicRegistrationEnabled: false,
    selfServiceAccountDeletionEnabled: false,
  })

  const isDirty = computed(() => {
    const currentSettings = settings.value
    if (!currentSettings) {
      return false
    }

    return (
      form.value.publicRegistrationEnabled !==
        currentSettings.publicRegistrationEnabled ||
      form.value.selfServiceAccountDeletionEnabled !==
        currentSettings.selfServiceAccountDeletionEnabled
    )
  })

  onMounted(() => {
    void loadSettings()
  })

  async function loadSettings() {
    loading.value = true

    try {
      const nextSettings = await getAdminSystemSettings()
      settings.value = nextSettings
      form.value = { ...nextSettings }
      loadFailed.value = false
    } catch (error) {
      loadFailed.value = true
      toast.error(
        getAPIErrorMessage(t, error, 'systemConfig.feedback.loadFailed'),
      )
    } finally {
      loading.value = false
    }
  }

  async function saveSettings() {
    if (!isDirty.value || saving.value) {
      return
    }

    saving.value = true
    const savePromise = updateSystemSettings({
      publicRegistrationEnabled: form.value.publicRegistrationEnabled,
      selfServiceAccountDeletionEnabled:
        form.value.selfServiceAccountDeletionEnabled,
    })

    toast.promise(savePromise, {
      loading: t('systemConfig.feedback.saving'),
      success: () => t('systemConfig.feedback.saved'),
      error: (error: unknown) =>
        getAPIErrorMessage(t, error, 'systemConfig.feedback.saveFailed'),
    })

    try {
      settings.value = await savePromise
      form.value = { ...settings.value }

      // These settings gate registration and self-service deletion, so the viewer's
      // capability set can change as a result of saving them. Refresh both, but never
      // let a refresh failure surface as a save failure — the save already succeeded.
      await authStore.refreshPublicState()
      await authStore.refreshViewer({ backgroundRequest: true })
    } catch (error) {
      // Rejection of `savePromise` is already reported by toast.promise above; catching
      // here keeps the awaited copy from becoming an unhandled rejection. The draft in
      // `form` is intentionally left untouched so the user can retry.
      console.warn('Failed to save or re-sync system settings', error)
    } finally {
      saving.value = false
    }
  }

  return {
    loading,
    saving,
    loadFailed,
    form,
    isDirty,
    loadSettings,
    saveSettings,
  }
}
