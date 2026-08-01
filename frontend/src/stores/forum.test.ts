import { createPinia, setActivePinia } from 'pinia'
import { http, HttpResponse } from 'msw'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { server } from '../test/setup'
import { useForumStore } from './forum'

describe('forum store notification reads', () => {
  beforeEach(() => {
    const storage = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
      clear: () => storage.clear(),
    })
    setActivePinia(createPinia())
  })

  it('keeps unread state and exposes feedback when marking a notification read fails', async () => {
    server.use(http.post('/api/v1/notifications/1/read', () => HttpResponse.json({
      error: { code: 'internal_error', message: 'read failed', requestId: 'store-notification-error' },
    }, { status: 500 })))

    const store = useForumStore()
    store.session = {
      user: {
        id: 7,
        publicId: 'u_7',
        email: 'student@example.com',
        nickname: '广东学生',
        role: 'student',
        province: '广东',
        grade: '高一',
        createdAt: '2026-07-31T00:00:00Z',
      },
      expiresAt: '2099-01-01T00:00:00Z',
    }
    store.notifications = [{
      id: 1,
      type: 'comment',
      title: '有人回复了你',
      summary: '查看新评论',
      targetUrl: '/posts/42',
      readAt: '',
      createdAt: '2026-07-31T00:00:00Z',
    }]

    await store.markNotificationsRead([1])

    expect(store.notifications[0].readAt).toBe('')
    expect(store.notificationReadError).toBe('通知状态同步失败，请稍后重试。')
  })

  it('opens login with a clear message when the current session expires', () => {
    const store = useForumStore()
    store.session = {
      user: {
        id: 8,
        publicId: 'u_8',
        email: 'expired@example.com',
        nickname: '过期用户',
        role: 'student',
        province: '广东',
        grade: '高一',
        createdAt: '2026-07-31T00:00:00Z',
      },
      expiresAt: '2020-01-01T00:00:00Z',
    }

    expect(store.requireAuth('/messages')).toBe(false)
    expect(store.authOpen).toBe(true)
    expect(store.authRedirect).toBe('/messages')
    expect(store.authMessage).toBe('登录状态已过期，请重新登录后继续。')
  })

})
