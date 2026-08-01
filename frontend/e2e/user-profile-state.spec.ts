import { expect, test } from '@playwright/test'

test('user profile shows a recoverable error state before profile data is available', async ({ page }) => {
  let profileAttempts = 0
  await page.route('**/api/v1/profiles/%E5%B9%BF%E4%B8%9C%E5%90%8C%E5%AD%A6', async (route) => {
    profileAttempts += 1
    if (profileAttempts === 1) {
      await route.fulfill({
        status: 500,
        json: { error: { code: 'internal_error', message: 'temporary profile failure', requestId: 'e2e-profile-error' } },
      })
      return
    }
    await route.fulfill({
      json: {
        data: {
          user: {
            id: 7,
            publicId: 'u_7',
            email: 'student@example.com',
            nickname: '广东同学',
            role: 'student',
            province: '广东',
            grade: '高一',
            createdAt: '2026-07-31T00:00:00Z',
          },
          bio: '',
          choiceProfile: {},
          stats: { posts: 0, comments: 0, following: 0, followers: 0, favorites: 0, engagement: 0 },
          posts: [],
          comments: [],
          favorites: [],
          following: [],
          followers: [],
          viewerFollowing: false,
        },
        meta: { requestId: 'e2e-profile-ok' },
      },
    })
  })

  await page.goto('/users/%E5%B9%BF%E4%B8%9C%E5%90%8C%E5%AD%A6')

  await expect(page.getByRole('heading', { name: '用户主页加载失败' })).toBeVisible()
  await expect(page.getByRole('button', { name: '重试' })).toBeVisible()

  await page.getByRole('button', { name: '重试' }).click()

  await expect(page.getByRole('heading', { name: '广东同学' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '用户主页加载失败' })).toBeHidden()
})
