import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { SystemSettings } from '@/lib/api/system-settings'
import { useSystemSettings } from '@/composables/system-settings/useSystemSettings'

const deps = vi.hoisted(() => ({
  getAdminSystemSettings: vi.fn(),
  updateSystemSettings: vi.fn(),
  refreshPublicState: vi.fn(),
  refreshViewer: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}))

// 只回显 key：断言落在最终选中的文案键上，与 error-messages.spec.ts 的做法一致。
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('vue-sonner', () => ({
  toast: { success: deps.toastSuccess, error: deps.toastError },
}))
vi.mock('@/lib/api/system-settings', () => ({
  getAdminSystemSettings: deps.getAdminSystemSettings,
  updateSystemSettings: deps.updateSystemSettings,
}))
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    refreshPublicState: deps.refreshPublicState,
    refreshViewer: deps.refreshViewer,
  }),
}))

const storedSettings: SystemSettings = {
  publicRegistrationEnabled: false,
  selfServiceAccountDeletionEnabled: false,
}

beforeEach(() => {
  vi.clearAllMocks()
  deps.getAdminSystemSettings.mockResolvedValue({ ...storedSettings })
  deps.updateSystemSettings.mockResolvedValue({ ...storedSettings })
  deps.refreshPublicState.mockResolvedValue(undefined)
  deps.refreshViewer.mockResolvedValue(null)
  // 组合式函数在组件外实例化时会注册 onMounted，Vue 会打一条“没有活动组件实例”的警告；
  // 本 spec 只验证状态机，静音它以免淹没真正的失败信息。
  vi.spyOn(console, 'warn').mockImplementation(() => undefined)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useSystemSettings', () => {
  it('loadSettings 采用服务端返回的策略', async () => {
    deps.getAdminSystemSettings.mockResolvedValue({
      publicRegistrationEnabled: true,
      selfServiceAccountDeletionEnabled: false,
    })

    const { form, loading, loadFailed, loadSettings } = useSystemSettings()
    await loadSettings()

    expect(form.value).toEqual({
      publicRegistrationEnabled: true,
      selfServiceAccountDeletionEnabled: false,
    })
    expect(loading.value).toBe(false)
    expect(loadFailed.value).toBe(false)
  })

  it('加载失败时置位 loadFailed 并提示加载失败文案', async () => {
    deps.getAdminSystemSettings.mockRejectedValue(new Error('offline'))

    const { loading, loadFailed, loadSettings } = useSystemSettings()
    await loadSettings()

    expect(loadFailed.value).toBe(true)
    expect(loading.value).toBe(false)
    expect(deps.toastError).toHaveBeenCalledWith(
      'systemConfig.feedback.loadFailed',
    )
  })

  it('只提交被切换的字段', async () => {
    deps.updateSystemSettings.mockResolvedValue({
      ...storedSettings,
      selfServiceAccountDeletionEnabled: true,
    })

    const { form, setSelfServiceAccountDeletionEnabled } = useSystemSettings()
    await setSelfServiceAccountDeletionEnabled(true)

    expect(deps.updateSystemSettings).toHaveBeenCalledWith({
      selfServiceAccountDeletionEnabled: true,
    })
    expect(form.value).toEqual({
      ...storedSettings,
      selfServiceAccountDeletionEnabled: true,
    })
    expect(deps.toastSuccess).toHaveBeenCalledWith(
      'systemConfig.feedback.saved',
    )
  })

  // 锁住一次真实回归：写入已落库后再刷新 viewer，刷新失败曾被外层 catch 当成“保存失败”，
  // 于是开关被回滚成旧值，还连续弹出“已保存”和“保存失败”两条相反的提示。
  it('写入成功后刷新失败不回滚、也不报保存失败', async () => {
    deps.updateSystemSettings.mockResolvedValue({
      ...storedSettings,
      publicRegistrationEnabled: true,
    })
    deps.refreshPublicState.mockRejectedValue(new Error('offline'))
    deps.refreshViewer.mockRejectedValue(new Error('offline'))

    const { form, saving, setPublicRegistrationEnabled } = useSystemSettings()
    await setPublicRegistrationEnabled(true)

    expect(form.value.publicRegistrationEnabled).toBe(true)
    expect(deps.toastSuccess).toHaveBeenCalledWith(
      'systemConfig.feedback.saved',
    )
    expect(deps.toastError).not.toHaveBeenCalled()
    // backgroundRequest 打的是“后台请求”标记：漏掉它，保存后这次 viewer 刷新会点亮顶部加载条。
    expect(deps.refreshViewer).toHaveBeenCalledWith({ backgroundRequest: true })
    expect(saving.value).toBe(false)
  })

  it('写入失败时回滚到写入前的值并提示保存失败文案', async () => {
    deps.updateSystemSettings.mockRejectedValue(new Error('conflict'))

    const { form, saving, setPublicRegistrationEnabled } = useSystemSettings()
    await setPublicRegistrationEnabled(true)

    expect(form.value).toEqual(storedSettings)
    expect(deps.toastError).toHaveBeenCalledWith(
      'systemConfig.feedback.saveFailed',
    )
    expect(deps.toastSuccess).not.toHaveBeenCalled()
    // catch 里 return 也必须经过 finally 释放锁，否则失败一次后开关永久禁用。
    expect(saving.value).toBe(false)
  })

  it('写入未返回前忽略第二次切换，只发一次请求', async () => {
    let release: ((value: SystemSettings) => void) | undefined
    deps.updateSystemSettings.mockReturnValue(
      new Promise<SystemSettings>((resolve) => {
        release = resolve
      }),
    )

    const { saving, setPublicRegistrationEnabled } = useSystemSettings()
    const first = setPublicRegistrationEnabled(true)
    await setPublicRegistrationEnabled(false)

    expect(deps.updateSystemSettings).toHaveBeenCalledTimes(1)
    expect(saving.value).toBe(true)

    release?.({ ...storedSettings, publicRegistrationEnabled: true })
    await first

    expect(saving.value).toBe(false)
    expect(deps.toastSuccess).toHaveBeenCalledTimes(1)
  })
})
