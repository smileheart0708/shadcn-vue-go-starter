import { computed, onScopeDispose, ref, watch } from 'vue'
import { useFileDialog, useObjectUrl } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { resolveErrorMessageOrInternal } from '@/lib/api/error-messages'
import { getUserRoleBadgeVariant, getUserRoleLabel } from '@/lib/auth/roles'
import { AVATAR_ACCEPTED_FILE_TYPES, getAvatarFallbackText } from '@/lib/avatar'
import { useAccountStore } from '@/stores/account'
import { useAuthStore } from '@/stores/auth'
import { normalizeOptionalEmail } from '@/utils/email'
import { compressAvatarFile, validateAvatarFile } from '@/utils/settings/avatar'

export function useProfileSettings() {
  const { t } = useI18n()
  const authStore = useAuthStore()
  const accountStore = useAccountStore()

  const editDialogOpen = ref(false)
  const editForm = ref({
    username: '',
    email: '',
  })
  const pendingAvatarFile = ref<File | null>(null)
  const profileError = ref('')
  const isSavingProfile = ref(false)
  const pendingAvatarPreviewURL = useObjectUrl(pendingAvatarFile)
  const avatarDialog = useFileDialog({
    accept: AVATAR_ACCEPTED_FILE_TYPES,
    multiple: false,
    reset: true,
  })

  const currentIdentity = computed(() => authStore.viewer?.identity ?? null)
  const currentRoleKey = computed(
    () => authStore.viewer?.authorization.role ?? null,
  )
  const avatarFallbackText = computed(() =>
    getAvatarFallbackText(currentIdentity.value?.username),
  )
  const avatarImageSrc = computed(
    () => currentIdentity.value?.avatarUrl ?? null,
  )
  const editAvatarFallbackText = computed(() => {
    const username = editForm.value.username.trim()
    return getAvatarFallbackText(
      username.length > 0 ? username : currentIdentity.value?.username,
    )
  })
  const editAvatarImageSrc = computed(
    () =>
      pendingAvatarPreviewURL.value ?? currentIdentity.value?.avatarUrl ?? null,
  )
  const roleLabel = computed(() => getUserRoleLabel(t, currentRoleKey.value))
  const roleBadgeVariant = computed(() =>
    getUserRoleBadgeVariant(currentRoleKey.value),
  )

  const stopAvatarChange = avatarDialog.onChange((files) => {
    const file = files?.[0]
    if (file === undefined) {
      return
    }

    void handleSelectedAvatarFile(file)
  })

  onScopeDispose(() => {
    stopAvatarChange.off()
  })

  watch(editDialogOpen, (isOpen) => {
    if (isOpen) {
      resetEditForm()
    }
  })

  function resetEditForm() {
    editForm.value = {
      username: currentIdentity.value?.username ?? '',
      email: currentIdentity.value?.email ?? '',
    }
    profileError.value = ''
    pendingAvatarFile.value = null
    avatarDialog.reset()
  }

  function openAvatarPicker() {
    avatarDialog.open()
  }

  async function handleSelectedAvatarFile(file: File) {
    const validation = validateAvatarFile(file)
    if (!validation.ok) {
      profileError.value =
        validation.failure === 'unsupported-type'
          ? t('settings.account.avatarUnsupportedType')
          : t('settings.account.avatarFileTooLarge')
      avatarDialog.reset()
      return
    }

    try {
      pendingAvatarFile.value = await compressAvatarFile(file)
      profileError.value = ''
    } catch {
      profileError.value = t('settings.account.avatarProcessFailed')
    } finally {
      avatarDialog.reset()
    }
  }

  async function saveProfile() {
    const username = editForm.value.username.trim()
    if (username.length === 0) {
      profileError.value = t('settings.account.usernameRequired')
      return
    }

    isSavingProfile.value = true
    profileError.value = ''

    try {
      await accountStore.saveProfile({
        username,
        email: normalizeOptionalEmail(editForm.value.email),
      })

      if (pendingAvatarFile.value !== null) {
        await accountStore.uploadAvatar(pendingAvatarFile.value)
        pendingAvatarFile.value = null
      }

      toast.success(t('settings.account.profileUpdated'))
      editDialogOpen.value = false
    } catch (error) {
      const message = resolveErrorMessageOrInternal(t, error)
      profileError.value = message
      toast.error(message)
    } finally {
      isSavingProfile.value = false
    }
  }

  return {
    currentIdentity,
    avatarFallbackText,
    avatarImageSrc,
    editDialogOpen,
    editForm,
    editAvatarFallbackText,
    editAvatarImageSrc,
    profileError,
    isSavingProfile,
    roleLabel,
    roleBadgeVariant,
    openAvatarPicker,
    saveProfile,
  }
}
