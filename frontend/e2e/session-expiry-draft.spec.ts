import { expect, test } from '@playwright/test'

const user = {
  id: 91,
  publicId: 'u_91',
  email: 'expired@example.com',
  nickname: '会话测试用户',
  role: 'student',
  province: '广东',
  grade: '高一',
  createdAt: '2026-07-31T00:00:00Z',
}

test('expired publish session asks for login without losing the draft', async ({ page }) => {
  const session = { user, expiresAt: new Date(Date.now() + 60 * 60 * 1000).toISOString() }
  await page.addInitScript((storedSession) => {
    localStorage.setItem('scf_auth_session', JSON.stringify(storedSession))
  }, session)

  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname
    if (path.endsWith('/posts') && request.method() === 'POST') {
      await route.fulfill({ status: 401, json: { error: { code: 'unauthorized', message: '会话已失效', requestId: 'expired-post' } } })
      return
    }
    if (path.endsWith('/auth/logout')) {
      await route.fulfill({ json: { data: { signedOut: true }, meta: { requestId: 'logout' } } })
      return
    }
    if (path.endsWith('/auth/login')) {
      await route.fulfill({ json: { data: session, meta: { requestId: 'login-again' } } })
      return
    }
    if (path.endsWith('/me/profile')) {
      await route.fulfill({ status: 503, json: { error: { code: 'UNAVAILABLE', message: '暂不可用', requestId: 'profile' } } })
      return
    }
    if (path.endsWith('/notifications')) {
      await route.fulfill({ json: { data: [], meta: { requestId: 'notifications', nextCursor: '', hasMore: false } } })
      return
    }
    await route.fulfill({ json: { data: [], meta: { requestId: 'public' } } })
  })

  await page.goto('/')
  await page.getByRole('button', { name: '发帖' }).click()
  await page.getByLabel('标题').fill('会话失效时保留这条标题')
  await page.getByLabel('正文').fill('这是一段在重新登录以后仍然需要保留的发帖正文。')
  await page.getByRole('button', { name: '发布', exact: true }).click()

  await expect(page.getByRole('dialog', { name: '登录账号' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '发布内容' })).toBeHidden()

  await page.getByLabel('邮箱').fill(user.email)
  await page.locator('input[autocomplete="current-password"]').fill('password123')
  await page.getByRole('button', { name: '登录并继续' }).click()

  await expect(page.getByRole('heading', { name: '发布内容' })).toBeVisible()
  await expect(page.getByLabel('标题')).toHaveValue('会话失效时保留这条标题')
  await expect(page.getByLabel('正文')).toHaveValue('这是一段在重新登录以后仍然需要保留的发帖正文。')
})
