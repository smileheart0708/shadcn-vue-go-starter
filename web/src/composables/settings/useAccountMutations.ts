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
 * These are plain calls, not Pinia state. Nothing here survives a route change and nothing is
 * shared between pages, so a store would only advertise state that does not exist — and
 * AGENTS.md §7 reserves `stores/` for state that is cross-route or persisted. The auth store
 * still owns the session; every write pushes the resulting viewer — or the terminated session —
 * back into it, so the dependency stays one-way: account writes -> auth.
 */
export function useAccountMutations() {
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
    // The server revoked every session for this user, so the local one is dead too.
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
}
