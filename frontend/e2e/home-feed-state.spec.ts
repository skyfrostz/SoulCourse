import { expect, test } from '@playwright/test'

test('home feed shows a retryable error instead of an empty discussion list', async ({ page }) => {
  let postAttempts = 0
  await page.route('**/api/v1/posts**', async (route) => {
    postAttempts += 1
    if (postAttempts === 1) {
      await route.fulfill({
        status: 503,
        json: { error: { code: 'service_unavailable', message: 'temporary feed failure', requestId: 'e2e-feed-error' } },
      })
      return
    }
    await route.fulfill({
      json: {
        data: [{
          id: 42,
          userId: 9,
          authorName: '广东同学',
          authorRole: 'student',
          title: '物化生如何选择专业',
          content: '想了解广东地区的专业覆盖情况。',
          imageUrls: [],
          tags: ['广东'],
          track: 'physics',
          electives: ['chemistry', 'biology'],
          category: 'question',
          grade: '高一',
          province: '广东',
          likesCount: 2,
          commentsCount: 0,
          favoritesCount: 1,
          viewerLiked: false,
          viewerFavorited: false,
          viewerFollowing: false,
          createdAt: '2026-07-31T00:00:00Z',
          updatedAt: '2026-07-31T00:00:00Z',
        }],
        meta: { requestId: 'e2e-feed-ok', hasMore: false },
      },
    })
  })
  await page.route('**/api/v1/insights', (route) => route.fulfill({ json: { data: [], meta: { requestId: 'e2e-home-insights' } } }))
  await page.route('**/api/v1/topics', (route) => route.fulfill({ json: { data: [], meta: { requestId: 'e2e-home-topics' } } }))

  await page.goto('/')

  await expect(page.getByRole('heading', { name: '讨论加载失败' })).toBeVisible()
  await expect(page.getByText('没有找到匹配的讨论')).toBeHidden()

  await page.getByRole('button', { name: '重试' }).click()

  await expect(page.getByRole('heading', { name: '物化生如何选择专业' })).toBeVisible()
})

test('home feed labels a failed request as offline after connectivity is lost', async ({ page, context }) => {
  await page.route('**/api/v1/posts**', (route) => route.abort('failed'))
  await page.route('**/api/v1/insights', (route) => route.fulfill({ json: { data: [], meta: { requestId: 'offline-insights' } } }))
  await page.route('**/api/v1/topics', (route) => route.fulfill({ json: { data: [], meta: { requestId: 'offline-topics' } } }))

  await page.goto('/')
  await expect(page.getByRole('heading', { name: '讨论加载失败' })).toBeVisible()
  await context.setOffline(true)
  await page.evaluate(() => window.dispatchEvent(new Event('offline')))

  await expect(page.getByRole('heading', { name: '当前网络不可用' })).toBeVisible()
  await expect(page.getByText('恢复网络后再重试，已发布讨论不会丢失。')).toBeVisible()
  await context.setOffline(false)
})
