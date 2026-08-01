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

test('following page shows a filtered empty state when search has no match', async ({ page }) => {
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
            stats: { posts: 0, comments: 0, following: 1, followers: 1, favorites: 0, engagement: 0 },
            posts: [],
            comments: [],
            favorites: [],
            viewerFollowing: false,
            following: [{
              name: '规划老师',
              role: 'teacher',
              province: '广东',
              grade: '高中',
              followedAt: '2026-07-31T00:00:00Z',
            }],
            followers: [{
              name: '学长',
              role: 'student',
              province: '广东',
              grade: '高二',
              followedAt: '2026-07-31T00:00:00Z',
            }],
          },
          meta: { requestId: 'following-profile' },
        },
      })
      return
    }

    if (url.pathname.endsWith('/notifications')) {
      await route.fulfill({
        json: {
          data: [{
            id: 1,
            type: 'comment',
            title: '有人回复了你',
            summary: '查看新评论',
            targetUrl: '/posts/42',
            createdAt: '2026-07-31T08:00:00Z',
          }],
          meta: { requestId: 'notifications-state' },
        },
      })
      return
    }

    await route.fulfill({ json: { data: {}, meta: { requestId: 'default-e2e' } } })
  })

  await page.goto('/following')
  await expect(page.getByRole('link', { name: /规划老师/ })).toBeVisible()
  await page.getByPlaceholder('搜索姓名、身份、省份...').fill('不存在')
  await expect(page.getByRole('heading', { name: '没有匹配的关注用户' })).toBeVisible()
  await expect(page.getByText('试试清空搜索词，或者换一个姓名、省份、身份关键词。')).toBeVisible()
})

test('notifications page shows a filtered empty state when search has no match', async ({ page }) => {
  await page.addInitScript(({ storedSession }) => {
    localStorage.setItem('scf_auth_session', JSON.stringify(storedSession))
  }, { storedSession: session })

  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (url.pathname.endsWith('/notifications')) {
      await route.fulfill({
        json: {
          data: [{
            id: 1,
            type: 'comment',
            title: '有人回复了你',
            summary: '查看新评论',
            targetUrl: '/posts/42',
            createdAt: '2026-07-31T08:00:00Z',
          }],
          meta: { requestId: 'notifications-state' },
        },
      })
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
            stats: { posts: 0, comments: 0, following: 1, followers: 1, favorites: 0, engagement: 0 },
            posts: [],
            comments: [],
            favorites: [],
            viewerFollowing: false,
            following: [{
              name: '规划老师',
              role: 'teacher',
              province: '广东',
              grade: '高中',
              followedAt: '2026-07-31T00:00:00Z',
            }],
            followers: [{
              name: '学长',
              role: 'student',
              province: '广东',
              grade: '高二',
              followedAt: '2026-07-31T00:00:00Z',
            }],
          },
          meta: { requestId: 'following-profile' },
        },
      })
      return
    }

    await route.fulfill({ json: { data: {}, meta: { requestId: 'default-e2e' } } })
  })

  await page.goto('/notifications')
  await expect(page.getByRole('heading', { name: '通知中心' })).toBeVisible()
  await expect(page.getByRole('button', { name: '评论互动', exact: true })).toBeVisible()
  await page.getByPlaceholder('搜索通知内容...').fill('不存在')
  const desktopEmptyState = page.locator('.notifications-shell > section.empty-state.compact-empty')
  await expect(desktopEmptyState.getByRole('heading', { name: '没有匹配的通知' })).toBeVisible()
  await expect(desktopEmptyState.getByText('试试清空搜索词，或者换一个通知关键词。')).toBeVisible()
})
