import { createPinia, setActivePinia } from 'pinia'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NotificationsPage from './NotificationsPage.vue'
import { useForumStore } from '../stores/forum'

const fetchNotifications = vi.fn()
const routerPush = vi.fn()
let pinia: ReturnType<typeof createPinia>

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return {
    ...actual,
    fetchNotifications: (query?: { limit?: number; cursor?: string }) => fetchNotifications(query),
  }
})

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush }),
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
}))

describe('NotificationsPage', () => {
  beforeEach(() => {
    fetchNotifications.mockReset()
    routerPush.mockReset()
    fetchNotifications.mockResolvedValue({
      items: [{
        id: 1,
        type: 'comment',
        title: '有人回复了你',
        summary: '查看新评论',
        targetUrl: '/posts/42',
        createdAt: '2026-07-31T00:00:00Z',
      }],
      hasMore: false,
    })
    const storage = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
      clear: () => storage.clear(),
    })
    pinia = createPinia()
    setActivePinia(pinia)
    const store = useForumStore()
    store.session = {
      user: {
        id: 1,
        publicId: 'u_1',
        email: 'student@example.com',
        nickname: '广东学生',
        role: 'student',
        province: '广东',
        grade: '高一',
        createdAt: '2026-07-31T00:00:00Z',
      },
      expiresAt: '2099-01-01T00:00:00Z',
    }
  })

  it('shows a filtered empty state when notification search has no match', async () => {
    render(NotificationsPage, {
      global: {
        plugins: [pinia, VueQueryPlugin],
        stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } },
      },
    })

    await waitFor(() => expect(screen.getAllByText('有人回复了你').length).toBeGreaterThan(0))
    await fireEvent.update(screen.getByPlaceholderText('搜索通知内容...'), '不存在')

    expect(screen.getByRole('heading', { name: '没有匹配的通知' })).toBeInTheDocument()
    expect(screen.getAllByText('试试清空搜索词，或者换一个通知关键词。').length).toBeGreaterThan(0)
  })

  it('loads the next notification page with the returned cursor', async () => {
    fetchNotifications
      .mockResolvedValueOnce({
        items: [{ id: 1, type: 'comment', title: '第一页通知', summary: '第一页', targetUrl: '/posts/1', createdAt: '2026-07-31T00:00:00Z' }],
        nextCursor: 'notification-page-2',
        hasMore: true,
      })
      .mockResolvedValueOnce({
        items: [{ id: 2, type: 'follow', title: '第二页通知', summary: '第二页', targetUrl: '/profiles/2', createdAt: '2026-07-30T00:00:00Z' }],
        hasMore: false,
      })

    render(NotificationsPage, {
      global: {
        plugins: [pinia, VueQueryPlugin],
        stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } },
      },
    })

    await waitFor(() => expect(screen.getAllByText('第一页通知').length).toBeGreaterThan(0))
    await fireEvent.click(screen.getAllByRole('button', { name: '加载更多' })[0])
    await waitFor(() => expect(screen.getAllByText('第二页通知').length).toBeGreaterThan(0))
    expect(fetchNotifications).toHaveBeenNthCalledWith(2, { limit: 30, cursor: 'notification-page-2' })
  })

  it('keeps loaded notifications visible when loading the next page fails', async () => {
    fetchNotifications
      .mockResolvedValueOnce({
        items: [{ id: 1, type: 'comment', title: '保留的通知', summary: '首屏内容', targetUrl: '/posts/1', createdAt: '2026-07-31T00:00:00Z' }],
        nextCursor: 'broken-page',
        hasMore: true,
      })
      .mockRejectedValueOnce(new Error('network error'))

    render(NotificationsPage, {
      global: {
        plugins: [pinia, VueQueryPlugin],
        stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } },
      },
    })

    await waitFor(() => expect(screen.getAllByText('保留的通知').length).toBeGreaterThan(0))
    await fireEvent.click(screen.getAllByRole('button', { name: '加载更多' })[0])
    await waitFor(() => expect(screen.getAllByText('更多通知加载失败，已加载内容不受影响。').length).toBeGreaterThan(0))
    expect(screen.getAllByText('保留的通知').length).toBeGreaterThan(0)
    expect(screen.getAllByRole('button', { name: '重试加载更多' }).length).toBeGreaterThan(0)
  })
})
