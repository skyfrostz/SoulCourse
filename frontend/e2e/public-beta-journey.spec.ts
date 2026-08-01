import { expect, test } from '@playwright/test'

const user = {
  id: 7,
  publicId: 'u_7',
  email: 'student@example.com',
  nickname: '广东学生',
  role: 'student',
  province: '广东',
  grade: '高一',
  createdAt: '2026-07-31T00:00:00Z',
}

const session = {
  user,
  token: 'e2e-session-token',
  expiresAt: '2099-01-01T00:00:00Z',
}

test('visitor can enter auth, sign in, and publish a post', async ({ page }) => {
  let uploadedObject = false
  const currentPost = {
    id: 42,
    userId: user.id,
    authorName: user.nickname,
    authorRole: user.role,
    title: '广东公测主链路',
    content: '这是一条用于验证公测发布流程的帖子。',
    imageUrls: ['/uploads/e2e/choice-proof.png'],
    tags: [] as string[],
    track: 'physics',
    electives: ['chemistry', 'biology'],
    category: 'question',
    grade: user.grade,
    province: user.province,
    likesCount: 0,
    commentsCount: 0,
    favoritesCount: 0,
    viewerLiked: false,
    viewerFavorited: false,
    viewerFollowing: false,
    createdAt: '2026-07-31T00:00:00Z',
    updatedAt: '2026-07-31T00:00:00Z',
  }
  await page.route('**/uploads/e2e/choice-proof.png', async (route) => {
    await route.fulfill({
      contentType: 'image/png',
      body: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAFgwJ/lFyW2QAAAABJRU5ErkJggg==', 'base64'),
    })
  })
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (url.pathname.endsWith('/auth/login')) {
      await route.fulfill({ json: { data: session, meta: { requestId: 'e2e-login' } } })
      return
    }
    if (url.pathname.endsWith('/me/profile')) {
      await route.fulfill({
        json: {
          data: {
            user,
            bio: '',
            choiceProfile: {
              realName: '',
              city: '',
              schoolType: '普通高中',
              gradeRank: '',
              mbti: '',
              targetMajors: '',
              targetCities: '',
              subjectStability: '中等',
              physicsScore: '',
              historyScore: '',
              chemistryScore: '',
              biologyScore: '',
              politicsScore: '',
              geographyScore: '',
              preferredTrack: 'physics',
              preferredSubjects: ['chemistry', 'biology'],
              learningStyle: '理解推导型',
              pressureTolerance: '中等',
              recommendationFocus: '专业覆盖率优先',
            },
            stats: { posts: 0, comments: 0, following: 0, followers: 0, favorites: 0, engagement: 0 },
            posts: [],
            comments: [],
            favorites: [],
            viewerFollowing: false,
            following: [],
            followers: [],
          },
          meta: { requestId: 'e2e-profile' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/notifications')) {
      await route.fulfill({ json: { data: [], meta: { requestId: 'e2e-notifications' } } })
      return
    }
    if (url.pathname.endsWith('/taxonomy')) {
      await route.fulfill({ json: { data: { topicTags: [] }, meta: { requestId: 'e2e-taxonomy' } } })
      return
    }
    if (url.pathname.endsWith('/uploads/images/presign') && request.method() === 'POST') {
      const payload = request.postDataJSON() as { fileName?: string; contentType?: string; sizeBytes?: number; width?: number; height?: number }
      expect(payload.fileName).toBe('choice-proof.png')
      expect(payload.contentType).toBe('image/png')
      expect(payload.width).toBe(1)
      expect(payload.height).toBe(1)
      expect(payload.sizeBytes).toBeGreaterThan(0)
      await route.fulfill({
        json: {
          data: {
            id: 'upload-e2e-image',
            assetKey: 'uploads/e2e/choice-proof.png',
            uploadUrl: '/api/v1/uploads/images/upload-e2e-image/object',
            method: 'PUT',
            contentType: 'image/png',
            maxBytes: 1024 * 1024,
            expiresAt: '2026-07-31T01:00:00Z',
          },
          meta: { requestId: 'e2e-upload-presign' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/uploads/images/upload-e2e-image/object') && request.method() === 'PUT') {
      uploadedObject = true
      await route.fulfill({ status: 204 })
      return
    }
    if (url.pathname.endsWith('/uploads/images/upload-e2e-image/complete') && request.method() === 'POST') {
      expect(uploadedObject).toBeTruthy()
      await route.fulfill({
        json: {
          data: {
            id: 'upload-e2e-image',
            assetKey: 'uploads/e2e/choice-proof.png',
            url: '/uploads/e2e/choice-proof.png',
            contentType: 'image/png',
            sizeBytes: 68,
            width: 1,
            height: 1,
          },
          meta: { requestId: 'e2e-upload-complete' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/posts') && request.method() === 'POST') {
      const payload = request.postDataJSON() as { imageUrls?: string[] }
      expect(payload.imageUrls).toEqual(['/uploads/e2e/choice-proof.png'])
      await route.fulfill({
        json: {
          data: currentPost,
          meta: { requestId: 'e2e-create-post' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/posts/42') && request.method() === 'PUT') {
      const payload = request.postDataJSON() as {
        title: string
        content: string
        tags: string[]
        track: string
        electives: string[]
        category: string
      }
      expect(payload).toMatchObject({
        title: '广东公测主链路更新',
        content: '这是一条已经编辑过的公测发布流程帖子。',
        tags: ['更新标签'],
        track: 'history',
        electives: ['politics', 'geography'],
        category: 'experience',
      })
      Object.assign(currentPost, {
        ...payload,
        updatedAt: '2026-07-31T00:20:00Z',
      })
      await route.fulfill({ json: { data: currentPost, meta: { requestId: 'e2e-update-post' } } })
      return
    }
    if (url.pathname.endsWith('/posts/42') && request.method() === 'DELETE') {
      await route.fulfill({ json: { data: { deleted: true }, meta: { requestId: 'e2e-delete-post' } } })
      return
    }
    if (url.pathname.endsWith('/posts/42') && request.method() === 'GET') {
      await route.fulfill({
        json: {
          data: {
            post: currentPost,
            comments: [],
          },
          meta: { requestId: 'e2e-post-detail' },
        },
      })
      return
    }
    if (url.pathname.includes('/posts') || url.pathname.endsWith('/insights') || url.pathname.endsWith('/topics')) {
      await route.fulfill({ json: { data: [], meta: { requestId: 'e2e-public-read' } } })
      return
    }
    await route.fulfill({ json: { data: {}, meta: { requestId: 'e2e-default' } } })
  })

  await page.goto('/')
  await page.getByRole('button', { name: '登录 / 注册' }).click()
  await expect(page.getByRole('dialog', { name: '登录账号' })).toBeVisible()

  await page.getByRole('button', { name: '注册', exact: true }).click()
  await expect(page.getByRole('dialog', { name: '注册账号' })).toBeVisible()
  await expect(page.getByLabel('省份')).toHaveValue('广东')

  await page.getByRole('button', { name: '登录', exact: true }).click()
  await page.getByLabel('邮箱').fill(user.email)
  await page.locator('input[autocomplete="current-password"]').fill('password123')
  await page.getByRole('button', { name: '登录并继续' }).click()

  await expect(page.getByRole('button', { name: /个人中心/ })).toBeVisible()
  await page.getByRole('button', { name: '发帖' }).click()
  await expect(page.getByRole('heading', { name: '发布内容' })).toBeVisible()

  await page.locator('input[type="file"]').setInputFiles({
    name: 'choice-proof.png',
    mimeType: 'image/png',
    buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAFgwJ/lFyW2QAAAABJRU5ErkJggg==', 'base64'),
  })
  await expect(page.getByRole('img', { name: '待发布图片预览' })).toBeVisible()

  await page.getByLabel('标题').fill('广东公测主链路')
  await page.getByLabel('正文').fill('这是一条用于验证公测发布流程的帖子。')
  await page.getByRole('button', { name: '发布', exact: true }).click()

  await expect(page).toHaveURL(/\/posts\/42$/)
  await page.getByRole('button', { name: '编辑帖子' }).click()
  await page.getByLabel('标题').fill('广东公测主链路更新')
  await page.getByLabel('正文').fill('这是一条已经编辑过的公测发布流程帖子。')
  await page.getByLabel('类型').selectOption('experience')
  await page.getByLabel('方向').selectOption('history')
  await page.getByRole('button', { name: '化学' }).click()
  await page.getByRole('button', { name: '生物' }).click()
  await page.getByRole('button', { name: '政治' }).click()
  await page.getByRole('button', { name: '地理' }).click()
  await page.getByLabel('标签').fill('更新标签')
  await page.getByRole('button', { name: '保存修改' }).click()
  await expect(page.getByRole('heading', { name: '广东公测主链路更新' })).toBeVisible()
  await expect(page.getByText('这是一条已经编辑过的公测发布流程帖子。')).toBeVisible()
  page.once('dialog', async (dialog) => {
    expect(dialog.message()).toContain('确认删除')
    await dialog.accept()
  })
  await page.getByRole('button', { name: '删除帖子' }).click()
  await expect(page).toHaveURL(/\/$/)
})

test('visitor can reset password from the auth modal', async ({ page }) => {
  let forgotRequested = false
  let resetRequested = false
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (url.pathname.endsWith('/auth/forgot-password') && request.method() === 'POST') {
      expect(request.postDataJSON()).toEqual({ email: user.email })
      forgotRequested = true
      await route.fulfill({
        json: {
          data: {
            sent: true,
            debugCode: '654321',
            retryAfterSeconds: 60,
            hourlyLimit: 5,
            hourlyRemaining: 4,
          },
          meta: { requestId: 'e2e-forgot-password' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/auth/reset-password') && request.method() === 'POST') {
      expect(request.postDataJSON()).toEqual({
        email: user.email,
        verificationCode: '654321',
        password: 'new-password123',
      })
      resetRequested = true
      await route.fulfill({ json: { data: { reset: true }, meta: { requestId: 'e2e-reset-password' } } })
      return
    }
    if (url.pathname.includes('/posts') || url.pathname.endsWith('/insights') || url.pathname.endsWith('/topics')) {
      await route.fulfill({ json: { data: [], meta: { requestId: 'e2e-public-read' } } })
      return
    }
    await route.fulfill({ json: { data: {}, meta: { requestId: 'e2e-default' } } })
  })

  await page.goto('/')
  await page.getByRole('button', { name: '登录 / 注册' }).click()
  await page.getByRole('button', { name: /忘记密码/ }).click()

  await expect(page.getByRole('dialog', { name: '重置密码' })).toBeVisible()
  await page.getByLabel('邮箱', { exact: true }).fill(user.email)
  await page.getByLabel('密码', { exact: true }).fill('new-password123')
  await page.getByLabel('确认新密码', { exact: true }).fill('new-password123')
  await page.getByRole('button', { name: '获取验证码' }).click()
  await expect(page.getByText(/本地调试验证码：654321/)).toBeVisible()
  await page.getByPlaceholder('6 位验证码').fill('654321')
  await page.getByRole('button', { name: '重置密码' }).click()

  await expect(page.getByRole('dialog', { name: '登录账号' })).toBeVisible()
  await expect(page.getByText('密码已重置，请使用新密码登录。')).toBeVisible()
  expect(forgotRequested).toBeTruthy()
  expect(resetRequested).toBeTruthy()
})
