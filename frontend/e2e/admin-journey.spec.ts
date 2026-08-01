import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'

async function clearBrowserState(page: Page) {
  await page.addInitScript(() => {
    localStorage.clear()
    sessionStorage.clear()
  })
}

async function mockPublicApi(page: Page) {
  await page.route('**/api/v1/posts**', (route) => route.fulfill({ json: { data: [], meta: { requestId: 'e2e-admin-public-posts' } } }))
  await page.route('**/api/v1/insights', (route) => route.fulfill({ json: { data: [], meta: { requestId: 'e2e-admin-public-insights' } } }))
  await page.route('**/api/v1/topics', (route) => route.fulfill({ json: { data: [], meta: { requestId: 'e2e-admin-public-topics' } } }))
}

async function mockAdminApi(
  page: Page,
  records: Array<Record<string, unknown>>,
  onWorkflow?: (requestBody: Record<string, unknown>) => Promise<Record<string, unknown>>,
  options: {
    reports?: Array<Record<string, unknown>>
    onModerateReport?: (requestBody: Record<string, unknown>, csrf: string | null) => Promise<void>
    onModerateUser?: (path: string, requestBody: Record<string, unknown>, csrf: string | null) => Promise<void>
  } = {},
) {
  await page.route('**/api/v1/admin/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (url.pathname.endsWith('/admin/login')) {
      const body = await request.postDataJSON()
      if (body.email !== 'admin@example.com' || body.password !== 'admin-password') {
        await route.fulfill({ status: 401, json: { error: { code: 'unauthorized', message: 'invalid credentials', requestId: 'e2e-admin-login-failed' } } })
        return
      }
      await route.fulfill({
        headers: { 'Set-Cookie': 'scf_admin_csrf=e2e-admin-csrf; Path=/; SameSite=Lax' },
        json: { data: { email: body.email, role: 'super_admin', permissions: ['dashboard.read', 'content.read', 'content.write', 'content.publish', 'content.delete', 'media.upload', 'moderation.read', 'moderation.act', 'users.read', 'users.ban', 'users.password_reset', 'system.email.read', 'system.email.test', 'audit.read'], expiresAt: '2099-01-01T00:00:00Z' }, meta: { requestId: 'e2e-admin-login' } },
      })
      return
    }
    if (url.pathname.endsWith('/admin/email-config')) {
      await route.fulfill({
        json: {
          data: {
            enabled: true,
            host: 'smtp.example.com',
            port: 465,
            usernameConfigured: true,
            passwordConfigured: true,
            fromEmail: 'noreply@example.com',
            replyTo: 'support@example.com',
            fromName: '选科π',
            useTLS: true,
            startTLS: false,
            emailVerificationTTLMinutes: 10,
            emailVerificationCooldownSeconds: 60,
            emailVerificationEmailHourlyLimit: 5,
            emailVerificationIPHourlyLimit: 20,
            emailVerificationMaxValidationAttempts: 5,
            missing: [],
          },
          meta: { requestId: 'e2e-admin-email-config' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/admin/content')) {
      await route.fulfill({
        json: {
          data: {
            records,
          },
          meta: { requestId: 'e2e-admin-content' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/admin/audit-logs')) {
      await route.fulfill({
        json: {
          data: {
            logs: [{
              action: 'publish_policy',
              recordId: 'policy-guangdong-2026',
              module: 'policies',
              detail: '管理员发布广东政策',
              actor: 'admin@example.com',
              createdAt: '2026-07-31T00:00:00Z',
            }],
          },
          meta: { requestId: 'e2e-admin-audit' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/admin/reports')) {
      await route.fulfill({ json: { data: { reports: options.reports || [] }, meta: { requestId: 'e2e-admin-reports' } } })
      return
    }
    if (/\/admin\/reports\/\d+\/moderate$/.test(url.pathname)) {
      const body = await request.postDataJSON()
      await options.onModerateReport?.(body, request.headers()['x-csrf-token'] || null)
      await route.fulfill({ json: { data: { id: Number(url.pathname.split('/').at(-2)), status: 'actioned' }, meta: { requestId: 'e2e-admin-moderate-report' } } })
      return
    }
    if (/\/admin\/users\/\d+\/(ban|restore)$/.test(url.pathname)) {
      const body = await request.postDataJSON()
      try {
        await options.onModerateUser?.(url.pathname, body, request.headers()['x-csrf-token'] || null)
      } catch (error) {
        await route.fulfill({ status: 500, json: { error: { code: 'user_moderation_failed', message: error instanceof Error ? error.message : 'user moderation failed' } } })
        return
      }
      await route.fulfill({ json: { data: { userId: Number(url.pathname.split('/').at(-2)) }, meta: { requestId: 'e2e-admin-moderate-user' } } })
      return
    }
    if (url.pathname.includes('/admin/content/') && url.pathname.endsWith('/workflow')) {
      const body = await request.postDataJSON()
      let nextRecord: Record<string, unknown> | undefined
      try {
        nextRecord = await onWorkflow?.(body)
      } catch (error) {
        if (error instanceof Error && error.message === 'unauthorized') {
          await route.fulfill({
            status: 401,
            json: {
              error: {
                code: 'unauthorized',
                message: 'invalid admin session',
                requestId: 'e2e-admin-workflow-unauthorized',
              },
            },
          })
          return
        }
        await route.fulfill({
          status: 500,
          json: {
            error: {
              code: 'workflow_save_failed',
              message: error instanceof Error ? error.message : 'workflow save failed',
              requestId: 'e2e-admin-workflow-failed',
            },
          },
        })
        return
      }
      await route.fulfill({
        json: {
          data: nextRecord || records[0],
          meta: { requestId: 'e2e-admin-workflow' },
        },
      })
      return
    }
    await route.fulfill({ json: { data: {}, meta: { requestId: 'e2e-admin-default' } } })
  })
}

async function signIn(page: Page) {
  await page.goto('/admin')
  await expect(page.getByRole('heading', { name: '选科π管理后台' })).toBeVisible()

  await page.getByPlaceholder('admin@example.com').fill('admin@example.com')
  await page.getByPlaceholder('请输入后台登录密码').fill('admin-password')
  await page.getByRole('button', { name: '登录后台' }).click()
  await expect(page.getByText('已连接后端内容库')).toBeVisible()
}

test('admin can sign in and sync remote content without a browser token', async ({ page }) => {
  await clearBrowserState(page)
  await mockPublicApi(page)
  await mockAdminApi(page, [
    {
      id: 'policy-guangdong-2026',
      module: 'policies',
      title: '广东 2026 选科政策',
      type: '政策文件',
      status: '已上架',
      scope: '广东',
      owner: '广东省教育考试院',
      tags: ['广东', '已复核'],
      summary: '广东公测政策样例。',
      url: 'https://example.com/policy.pdf',
      priority: '高',
      sortOrder: 1,
      payload: {},
      updatedAt: '2026-07-31T00:00:00Z',
    },
  ])

  await signIn(page)
  await expect(page.getByText('内容总量')).toBeVisible()
  await page.getByRole('button', { name: /政策库/ }).click()
  await expect(page.getByText('广东 2026 选科政策')).toBeVisible()
})

test('admin stale browser session is cleared when backend rejects it', async ({ page }) => {
  await mockPublicApi(page)
  await page.addInitScript(() => {
    localStorage.setItem('scf_admin_session', JSON.stringify({
      authenticated: true,
      mode: 'online',
      email: 'admin@example.com',
      signedAt: '2026-07-30T00:00:00Z',
    }))
  })
  await page.route('**/api/v1/admin/email-config', (route) => route.fulfill({
    status: 401,
    json: { error: { code: 'unauthorized', message: 'invalid admin session', requestId: 'e2e-admin-stale' } },
  }))

  await page.goto('/admin')

  await expect(page.getByRole('heading', { name: '选科π管理后台' })).toBeVisible()
  await expect(page.getByRole('button', { name: '登录后台' })).toBeVisible()
  await expect(page.getByText('已连接后端内容库')).toBeHidden()
  await expect.poll(() => page.evaluate(() => localStorage.getItem('scf_admin_session'))).toBeNull()
})

test('admin can review a report, retry hide failure, and restore content', async ({ page }) => {
  await clearBrowserState(page)
  await mockPublicApi(page)

  const reportedPost = {
    id: 'post-report-1',
    module: 'posts',
    title: '被举报的选科经验帖',
    type: '经验帖',
    status: '已上架',
    scope: '广东',
    owner: '社区审核',
    tags: ['举报', '物化生'],
    summary: '用户举报该帖包含误导性选科建议。',
    url: '/posts/101',
    priority: '高',
    sortOrder: 1,
    payload: {
      reportReason: '误导性建议',
      reportDetail: '帖子建议忽略目标专业选科要求，需要管理员核验。',
      reporterName: '广东高一用户',
      reportCount: 2,
      reportedAt: '2026-07-31T08:30:00Z',
      workflow: [],
    },
    updatedAt: '2026-07-31T08:30:00Z',
  }
  let hideAttempts = 0

  await mockAdminApi(page, [reportedPost], async (body) => {
    if (body.action === 'unpublish-content') {
      hideAttempts += 1
      await new Promise((resolve) => setTimeout(resolve, 600))
      if (hideAttempts === 1) {
        throw new Error('workflow save failed')
      }
      return {
        ...reportedPost,
        status: '下架',
        priority: '高',
        payload: {
          ...reportedPost.payload,
          workflow: [{ time: '2026-07-31 16:40', action: '隐藏内容', from: '已上架', to: '下架', note: body.note, actor: 'admin' }],
        },
        updatedAt: '2026-07-31 16:40',
      }
    }
    await new Promise((resolve) => setTimeout(resolve, 600))
    return {
      ...reportedPost,
      status: '待审核',
      priority: '中',
      payload: {
        ...reportedPost.payload,
        workflow: [
          { time: '2026-07-31 16:40', action: '隐藏内容', from: '已上架', to: '下架', note: '核验举报属实，先隐藏。', actor: 'admin' },
          { time: '2026-07-31 16:45', action: '恢复展示审核', from: '下架', to: '待审核', note: body.note, actor: 'admin' },
        ],
      },
      updatedAt: '2026-07-31 16:45',
    }
  })

  await signIn(page)
  await page.getByRole('button', { name: /帖子管理/ }).click()
  await page.getByText('被举报的选科经验帖').click()

  await expect(page.getByLabel('举报详情')).toContainText('误导性建议')
  await expect(page.getByLabel('举报详情')).toContainText('广东高一用户')

  await page.getByRole('button', { name: '隐藏内容' }).click()
  await page.getByPlaceholder('请写明举报核验结果、隐藏原因和恢复条件。').fill('核验举报属实，先隐藏。')
  await page.getByRole('button', { name: '确认执行' }).click()
  await expect(page.getByRole('alert')).toContainText('workflow save failed')

  await page.getByRole('button', { name: '确认执行' }).click()
  await expect(page.getByRole('status')).toHaveText('隐藏内容已完成')
  await expect(page.locator('.workflow-head .status')).toHaveText('下架')

  await page.getByRole('button', { name: '恢复展示审核' }).click()
  await page.getByPlaceholder('说明误报依据、整改结果或恢复展示条件。').fill('作者已整改，恢复到审核队列。')
  await page.getByRole('button', { name: '确认执行' }).click()
  await expect(page.getByRole('status')).toHaveText('恢复展示审核已完成')
  await expect(page.locator('.workflow-head .status')).toHaveText('待审核')
})

test('admin workflow 401 clears stale session before showing more controls', async ({ page }) => {
  await clearBrowserState(page)
  await mockPublicApi(page)

  const reportedPost = {
    id: 'post-report-stale',
    module: 'posts',
    title: '会话过期时的举报帖',
    type: '经验帖',
    status: '已上架',
    scope: '广东',
    owner: '社区审核',
    tags: ['举报'],
    summary: '管理员点击审核动作时后端会话已经失效。',
    url: '/posts/102',
    priority: '高',
    sortOrder: 1,
    payload: {
      reportReason: '疑似误导',
      reportDetail: '需要管理员核验后再处理。',
      reporterName: '广东用户',
      reportCount: 1,
      reportedAt: '2026-07-31T09:00:00Z',
      workflow: [],
    },
    updatedAt: '2026-07-31T09:00:00Z',
  }

  await mockAdminApi(page, [reportedPost], async () => {
    throw new Error('unauthorized')
  })

  await signIn(page)
  await page.getByRole('button', { name: /帖子管理/ }).click()
  await page.getByText('会话过期时的举报帖').click()
  await page.getByRole('button', { name: '隐藏内容' }).click()
  await page.getByPlaceholder('请写明举报核验结果、隐藏原因和恢复条件。').fill('会话过期时尝试隐藏。')
  await page.getByRole('button', { name: '确认执行' }).click()

  await expect(page.getByRole('heading', { name: '选科π管理后台' })).toBeVisible()
  await expect(page.getByText('后台会话已失效，请重新登录')).toBeVisible()
  await expect.poll(() => page.evaluate(() => localStorage.getItem('scf_admin_session'))).toBeNull()
})

test('real report queue uses moderation endpoint with admin csrf', async ({ page }) => {
  await clearBrowserState(page)
  await mockPublicApi(page)
  let moderationRequest: { body: Record<string, unknown>; csrf: string | null } | null = null
  await mockAdminApi(page, [], undefined, {
    reports: [{
      id: 71,
      reporterId: 9,
      reporterName: '公测用户',
      targetType: 'post',
      targetId: 301,
      targetTitle: '真实举报队列帖子',
      targetAuthor: '待核验作者',
      reason: '疑似误导',
      detail: '专业要求与官方来源不一致。',
      status: 'open',
      resolutionNote: '',
      createdAt: '2026-07-31T10:00:00Z',
      updatedAt: '2026-07-31T10:00:00Z',
    }],
    onModerateReport: async (body, csrf) => { moderationRequest = { body, csrf } },
  })

  await signIn(page)
  await page.getByRole('button', { name: /帖子管理/ }).click()
  await page.getByText('真实举报队列帖子').click()
  await expect(page.getByLabel('举报详情')).toContainText('专业要求与官方来源不一致')
  await page.getByRole('button', { name: '隐藏内容' }).click()
  await page.getByPlaceholder('请写明举报核验结果和隐藏原因。').fill('已核对官方来源，内容存在误导。')
  await page.getByRole('button', { name: '确认执行' }).click()

  await expect(page.locator('.workflow-head .status')).toHaveText('下架')
  expect(moderationRequest).toEqual({
    body: { action: 'hide', note: '已核对官方来源，内容存在误导。' },
    csrf: 'e2e-admin-csrf',
  })
})

test('user freeze calls account ban endpoint and rolls back on failure', async ({ page }) => {
  await clearBrowserState(page)
  await mockPublicApi(page)
  const userRecord = {
    id: 'user-42', module: 'users', title: '风险账号', type: '学生', status: '正常', scope: '广东', owner: '账号系统',
    tags: ['风险关注'], summary: '多次发布误导内容。', url: '', priority: '高', sortOrder: 0,
    payload: { userId: 42, email: 'risk@example.com', postCount: 3, passwordConfigured: true }, updatedAt: '2026-07-31T10:00:00Z',
  }
  await mockAdminApi(page, [userRecord], undefined, {
    onModerateUser: async (path, body, csrf) => {
      expect(path).toContain('/admin/users/42/ban')
      expect(body).toEqual({ reason: '多次发布误导内容，公测期间暂停账号。' })
      expect(csrf).toBe('e2e-admin-csrf')
      throw new Error('ban persistence failed')
    },
  })

  await signIn(page)
  await page.getByRole('button', { name: /用户与权限/ }).click()
  await page.getByText('风险账号').click()
  await page.getByRole('button', { name: '冻结账号' }).click()
  await page.getByPlaceholder('请写明冻结原因和恢复条件。').fill('多次发布误导内容，公测期间暂停账号。')
  await page.getByRole('button', { name: '确认执行' }).click()

  await expect(page.getByRole('alert')).toContainText('ban persistence failed')
  await expect(page.locator('.permission-head .status')).toHaveText('正常')
})
