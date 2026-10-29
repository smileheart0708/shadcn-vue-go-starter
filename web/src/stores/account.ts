import { defineStore } from 'pinia'
import type { ChangePasswordInput, UpdateProfileInput } from '@/lib/api/account'
import {
  deleteAccount as deleteAccountRequest,
  updatePassword as updatePasswordRequest,
  updateProfile as updateProfileRequest,
  uploadAvatar as uploadAvatarRequest,
} from '@/lib/api/account'
import { useAuthStore } from '@/stores/auth'

/**
 * Account self-service writes (profile, avatar, password, deletion).
 *
 * The auth store owns the session; this store only mutates the account and then
 * pushes the resulting viewer — or a terminated session — back into it. The
 * dependency is one-way: account -> auth.
 */
export const useAccountStore = defineStore('account', () => {
  const authStore = useAuthStore()

  async function saveProfile(profile: UpdateProfileInput) {
    const nextViewer = await updateProfileRequest(profile)
    authStore.applyViewer(nextViewer)
    return nextViewer
  }

  async function uploadAvatar(file: File) {
    const nextViewer = await uploadAvatarRequest(file)
    authStore.applyViewer(nextViewer)
    return nextViewer
  }

  async function changePassword(input: ChangePasswordInput) {
    await updatePasswordRequest(input)
    authStore.resetSession(true)
  }

  async function deleteAccount() {
    await deleteAccountRequest()
    authStore.resetSession(true)
  }

  return {
    saveProfile,
    uploadAvatar,
    changePassword,
    deleteAccount,
  }
})
