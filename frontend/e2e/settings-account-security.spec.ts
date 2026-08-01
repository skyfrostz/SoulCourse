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

test('signed-in user can review and revoke another account session from settings', async ({ page }) => {
  let revokedSession = false

  await page.addInitScript(({ storedSession }) => {
    localStorage.setItem('scf_auth_session', JSON.stringify(storedSession))
  }, { storedSession: session })

  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())

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
          meta: { requestId: 'settings-profile' },
        },
      })
      return
    }

    if (url.pathname.endsWith('/notifications')) {
      await route.fulfill({ json: { data: [], meta: { requestId: 'settings-notifications' } } })
      return
    }

    if (url.pathname.endsWith('/me/sessions') && request.method() === 'GET') {
      await route.fulfill({
        json: {
          data: [
            { id: 101, createdAt: '2026-07-31T08:00:00Z', expiresAt: '2026-08-01T08:00:00Z', current: true },
            { id: 102, createdAt: '2026-07-30T18:30:00Z', expiresAt: '2026-08-01T18:30:00Z', current: false },
          ].filter((item) => !revokedSession || item.id !== 102),
          meta: { requestId: 'settings-sessions' },
        },
      })
      return
    }

    if (url.pathname.endsWith('/me/sessions/102') && request.method() === 'DELETE') {
      revokedSession = true
      await route.fulfill({ json: { data: { revoked: true }, meta: { requestId: 'settings-revoke-session' } } })
      return
    }

    await route.fulfill({ json: { data: {}, meta: { requestId: 'settings-default' } } })
  })

  await page.goto('/settings')

  await expect(page.getByRole('heading', { name: '账号安全' })).toBeVisible()
  await expect(page.getByText('当前设备')).toBeVisible()
  await expect(page.getByText('设备会话 #102')).toBeVisible()

  await page.getByRole('button', { name: '撤销' }).click()

  await expect(page.getByText('已撤销该设备会话。')).toBeVisible()
  await expect(page.getByText('设备会话 #102')).toBeHidden()
})
