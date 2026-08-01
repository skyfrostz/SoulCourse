import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { createPinia, setActivePinia } from 'pinia'
import { http, HttpResponse, delay } from 'msw'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { server } from '../test/setup'
import { useForumStore } from '../stores/forum'
import AuthModal from './AuthModal.vue'

async function renderModal() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div />' } }],
  })
  await router.push('/')
  await router.isReady()
  const store = useForumStore()
  store.authOpen = true
  render(AuthModal, { global: { plugins: [pinia, router] } })
  return store
}

describe('AuthModal public beta states', () => {
  beforeEach(() => {
    const storage = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
      clear: () => storage.clear(),
    })
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: true })
    localStorage.clear()
  })

  it('prevents duplicate login submissions while the request is pending', async () => {
    let requests = 0
    server.use(http.post('/api/v1/auth/login', async () => {
      requests += 1
      await delay(80)
      return HttpResponse.json({
        data: {
          user: { id: 1, publicId: 'u_1', email: 'user@example.com', nickname: '公测用户', role: 'student', province: '广东', grade: '高一', createdAt: '2026-07-31T00:00:00Z' },
          expiresAt: '2099-01-01T00:00:00Z',
        },
        meta: { requestId: 'login-test' },
      })
    }))
    await renderModal()

    await fireEvent.update(screen.getByLabelText('邮箱'), 'user@example.com')
    await fireEvent.update(screen.getByLabelText('密码'), 'password123')
    const submit = screen.getByRole('button', { name: '登录并继续' })
    await fireEvent.click(submit)

    expect(screen.getByRole('button', { name: '处理中...' })).toBeDisabled()
    expect(screen.getByRole('button', { name: '关闭' })).toBeDisabled()
    await fireEvent.click(screen.getByRole('button', { name: '处理中...' }))
    await waitFor(() => expect(requests).toBe(1))
  })

  it('shows an offline state and does not send a login request', async () => {
    const loginSpy = vi.fn()
    server.use(http.post('/api/v1/auth/login', loginSpy))
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: false })

    await renderModal()

    expect(screen.getByText('当前网络不可用，恢复网络后可继续。')).toBeVisible()
    expect(screen.getByRole('button', { name: '登录并继续' })).toBeDisabled()
    expect(loginSpy).not.toHaveBeenCalled()
  })
})
