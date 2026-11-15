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

// 手动控制 settle 时机，才能构造“先发出的请求后返回”这个顺序。
function deferred<T>() {
  let resolve: (value: T) => void = () => undefined
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise
  })
  return { promise, resolve }
}

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

  // 回归：翻页请求与筛选提交并发时，先发出的响应后到会把 page 写回旧值，
  // 表格停在第 2 页而筛选框显示的是新条件。
  it('并发请求只让最新一次写入状态', async () => {
    const stale = deferred<typeof firstPage>()
    const fresh = deferred<typeof firstPage>()
    deps.listAdminUsers
      .mockReturnValueOnce(stale.promise)
      .mockReturnValueOnce(fresh.promise)

    const { users, page, total, refreshing, loadUsers } = useAdminUsersList()

    page.value = 2
    const staleRun = loadUsers({ background: true })

    page.value = 1
    const freshRun = loadUsers()

    fresh.resolve({ items: [managedUser], page: 1, pageSize: 10, total: 1 })
    await freshRun

    stale.resolve({ items: [], page: 2, pageSize: 10, total: 1 })
    await staleRun

    expect(page.value).toBe(1)
    expect(users.value).toEqual([managedUser])
    expect(total.value).toBe(1)
    expect(refreshing.value).toBe(false)
  })

  it('加载失败置位 loadFailed、提示文案，并复位两个加载标记', async () => {
    deps.listAdminUsers.mockRejectedValue(new Error('offline'))

    const { loading, refreshing, loadFailed, loadUsers } = useAdminUsersList()
    await loadUsers({ background: true })

    expect(loadFailed.value).toBe(true)
    expect(loading.value).toBe(false)
    expect(refreshing.value).toBe(false)
    expect(deps.toastError).toHaveBeenCalledWith('common.feedback.loadFailed')
  })

  // 回归：翻页曾经先写 page 再请求，失败后页码已经前进、表格还是旧数据，而失败态会把
  // 分页条摘掉 —— 用户既回不到上一页也没有重试入口，只能整页刷新。现在目标页只作为
  // 请求参数，成功响应才提交。
  it('翻页失败时页码停在原页，请求仍指向目标页', async () => {
    deps.listAdminUsers
      .mockResolvedValueOnce({
        items: [managedUser],
        page: 1,
        pageSize: 10,
        total: 25,
      })
      .mockRejectedValueOnce(new Error('offline'))

    const { page, loadFailed, loadUsers, goToNextPage } = useAdminUsersList()
    await loadUsers()

    goToNextPage()
    await vi.waitFor(() => {
      expect(loadFailed.value).toBe(true)
    })

    expect(deps.listAdminUsers).toHaveBeenLastCalledWith(
      expect.objectContaining({ page: 2 }),
    )
    expect(page.value).toBe(1)
  })
})
