import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { resolveAPIErrorMessage } from '@/lib/api/error-messages'
import {
  getAdminSystemSettings,
  updateSystemSettings,
  type SystemSettings,
  type UpdateSystemSettingsInput,
} from '@/lib/api/system-settings'
import { useAuthStore } from '@/stores/auth'

export function useSystemSettings() {
  const { t } = useI18n()
  const authStore = useAuthStore()

  const loading = ref(true)
  const loadFailed = ref(false)
  // 两个开关共用同一个 PATCH 端点：请求未返回前锁住整组，避免两次写入互相覆盖。
  const saving = ref(false)
  // null 表示「还没拿到服务端值」。失败时保持 null，页面据此把值位置渲染成骨架，
  // 而不是显示一个编造的 false 被当成真实策略读走。
  const form = ref<SystemSettings | null>(null)

  onMounted(() => {
    void loadSettings()
  })

  async function loadSettings() {
    loading.value = true

    try {
      form.value = await getAdminSystemSettings()
      loadFailed.value = false
    } catch (error) {
      loadFailed.value = true
      toast.error(
        resolveAPIErrorMessage(t, error) ??
          t('systemConfig.feedback.loadFailed'),
      )
    } finally {
      loading.value = false
    }
  }

  async function applyChange(patch: UpdateSystemSettingsInput) {
    if (saving.value) {
      return
    }

    // 未加载成功时页面只有骨架、没有可点的开关；守卫是为了让 null 不会被当成可写状态。
    const current = form.value
    if (current === null) {
      return
    }

    // 开关要立刻反映点击结果，所以先写本地；失败时回滚到写入前的快照。
    const previous = { ...current }
    form.value = { ...current, ...patch }
    saving.value = true

    try {
      // 只提交被切换的字段：另一个字段即使被其他管理员改过，本次写入也不会覆盖它。
      form.value = await updateSystemSettings(patch)
      toast.success(t('systemConfig.feedback.saved'))
    } catch (error) {
      form.value = previous
      toast.error(
        resolveAPIErrorMessage(t, error) ??
          t('systemConfig.feedback.saveFailed'),
      )
      return
    } finally {
      saving.value = false
    }

    // 走到这里说明写入已经成功。下面两个刷新只影响本地 viewer 与注册开关状态，失败
    // 不能回滚 form、更不能报“保存失败”——否则用户会同时看到“已保存”和“保存失败”，
    // 而服务端其实已经改了。旧实现只有一个 catch，重构时最容易在这里退回去。
    const resynced = await Promise.allSettled([
      authStore.refreshPublicState(),
      authStore.refreshViewer({ backgroundRequest: true }),
    ])
    const rejected = resynced.filter((result) => result.status === 'rejected')
    if (rejected.length > 0) {
      console.warn(
        'Failed to re-sync viewer after saving system settings',
        rejected,
      )
    }
  }

  // 返回写入 Promise 只是为了让调用方（和 spec）能等到落库结果；模板里直接绑定即可。
  async function setPublicRegistrationEnabled(value: boolean) {
    return applyChange({ publicRegistrationEnabled: value })
  }

  async function setSelfServiceAccountDeletionEnabled(value: boolean) {
    return applyChange({ selfServiceAccountDeletionEnabled: value })
  }

  return {
    loading,
    saving,
    loadFailed,
    form,
    loadSettings,
    setPublicRegistrationEnabled,
    setSelfServiceAccountDeletionEnabled,
  }
}
