import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { ManagedUser } from '@/lib/api/admin-users'
import { useAdminUsersList } from '@/composables/admin-users/useAdminUsersList'

const deps = vi.hoisted(() => ({
  listAdminUsers: vi.fn(),
  toastError: vi.fn(),
}))

// 只回显 key：断言落在最终选中的文案键上，与 error-messages.spec.ts 的做法一致。
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('vue-sonner', () => ({ toast: { error: deps.toastError } }))
vi.mock('@/lib/api/admin-users', () => ({
  listAdminUsers: deps.listAdminUsers,
}))

const managedUser: ManagedUser = {
  id: 1,
  username: 'owner',
  email: 'owner@example.com',
  avatarUrl: null,
  role: 'owner',
  status: 'active',
  lastActiveAt: null,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
  actions: [],
}

const firstPage = {
  items: [managedUser],
  page: 1,
  pageSize: 10,
  total: 3,
}

beforeEach(() => {
  vi.clearAllMocks()
  deps.listAdminUsers.mockResolvedValue({ ...firstPage })
  // 组合式函数在组件外实例化时会注册 onMounted，Vue 会打一条“没有活动组件实例”的警告；
  // 本 spec 只验证状态机，静音它以免淹没真正的失败信息。
  vi.spyOn(console, 'warn').mockImplementation(() => undefined)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useAdminUsersList', () => {
  it('加载成功写入列表与分页信息', async () => {
    const { users, page, pageSize, total, loadFailed, loadUsers } =
      useAdminUsersList()
    await loadUsers()

    expect(users.value).toEqual(firstPage.items)
    expect(page.value).toBe(1)
    expect(pageSize.value).toBe(10)
    expect(total.value).toBe(3)
    expect(loadFailed.value).toBe(false)
  })

  // 骨架与分页条依赖这条契约：背景刷新只能动 refreshing，不能把 loading 打开，
  // 否则已有数据时表格会退回首屏骨架。
  it('背景刷新只置 refreshing，不打开 loading', async () => {
    const { loading, refreshing, loadUsers } = useAdminUsersList()
    await loadUsers()
    expect(loading.value).toBe(false)

    const pending = loadUsers({ background: true })
    expect(refreshing.value).toBe(true)
    expect(loading.value).toBe(false)

    await pending
    expect(refreshing.value).toBe(false)
  })

  it('加载失败置位 loadFailed、提示文案，并复位两个加载标记', async () => {
    deps.listAdminUsers.mockRejectedValue(new Error('offline'))

    const { loading, refreshing, loadFailed, loadUsers } = useAdminUsersList()
    await loadUsers({ background: true })

    expect(loadFailed.value).toBe(true)
    expect(loading.value).toBe(false)
    expect(refreshing.value).toBe(false)
    expect(deps.toastError).toHaveBeenCalledWith(
      'adminUsers.feedback.loadFailed',
    )
  })
})
