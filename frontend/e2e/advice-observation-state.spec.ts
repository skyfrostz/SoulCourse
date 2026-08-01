import { expect, test } from '@playwright/test'

const post = {
  id: 73,
  authorName: '广东学生',
  authorRole: 'student',
  title: '物化生组合如何核对专业要求',
  content: '先用官方目录核对目标专业，再评估成绩。',
  imageUrls: [],
  tags: ['物化生'],
  track: 'physics',
  electives: ['chemistry', 'biology'],
  category: 'question',
  grade: '高一',
  province: '广东',
  likesCount: 3,
  commentsCount: 1,
  favoritesCount: 1,
  viewerLiked: false,
  viewerFavorited: false,
  viewerFollowing: false,
  createdAt: '2026-07-31T00:00:00Z',
  updatedAt: '2026-07-31T00:00:00Z',
}

test.describe('advice and observation public-beta states', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/posts**', async (route) => {
      await route.fulfill({ json: { data: [post], meta: { requestId: 'advice-posts', hasMore: false } } })
    })
    await page.route('**/api/v1/taxonomy', async (route) => {
      await route.fulfill({
        json: {
          data: { topicTags: [], subjectTags: [{ value: '物化生', label: '物化生' }] },
          meta: { requestId: 'advice-taxonomy' },
        },
      })
    })
  })

  test('shows real advice posts, truthful search empty state, and redirects legacy detail links', async ({ page }) => {
    await page.goto('/advice')
    await expect(page.getByRole('heading', { name: '同一帖子库里的选科经验' })).toBeVisible()
    await expect(page.locator('.topic-feed-panel').getByRole('heading', { name: post.title })).toBeVisible()

    await page.getByPlaceholder('搜索组合、专业或省份').fill('不存在的组合')
    await expect(page.getByRole('heading', { name: '没有匹配的建议帖子' })).toBeVisible()

    await page.goto('/advice/73')
    await expect(page).toHaveURL(/\/posts\/73$/)
  })

  test('does not disguise a taxonomy failure as an empty advice list', async ({ page }) => {
    await page.unroute('**/api/v1/taxonomy')
    await page.route('**/api/v1/taxonomy', async (route) => {
      await route.fulfill({ status: 503, json: { error: { code: 'unavailable', message: '暂时不可用', requestId: 'taxonomy-error' } } })
    })
    await page.goto('/advice')
    await expect(page.getByRole('heading', { name: '选科建议暂时无法加载' })).toBeVisible()
    await expect(page.getByRole('button', { name: '重新加载' })).toBeVisible()
  })

  test('labels topic and advice failures as offline when the browser loses connectivity', async ({ page, context }) => {
    await page.route('**/api/v1/topics', async (route) => {
      await route.abort('failed')
    })
    await page.goto('/topics')
    await context.setOffline(true)
    await page.evaluate(() => window.dispatchEvent(new Event('offline')))
    await expect(page.getByRole('heading', { name: '当前网络不可用' })).toBeVisible()

    await context.setOffline(false)
    await page.unroute('**/api/v1/topics')
    await page.route('**/api/v1/taxonomy', async (route) => {
      await route.abort('failed')
    })
    await page.route('**/api/v1/posts**', async (route) => {
      await route.abort('failed')
    })
    await page.goto('/advice')
    await context.setOffline(true)
    await page.evaluate(() => window.dispatchEvent(new Event('offline')))
    await expect(page.getByRole('heading', { name: '当前网络不可用' })).toBeVisible()
    await context.setOffline(false)
  })

  test('shows published observation data and a recoverable API failure state', async ({ page }) => {
    await page.route('**/api/v1/insights', async (route) => {
      await route.fulfill({
        json: {
          data: [{
            id: 5,
            combination: '物理+化学',
            trend: '稳定',
            heat: 630,
            matchRate: 82,
            advice: '优先核对目标专业要求。',
            details: '广东已复核数据。',
            metricType: '计划数',
            unit: '个',
            province: '广东',
            dataYear: 2026,
            sourceName: '广东考试院',
            sourceUrl: 'https://example.test/insight',
            scope: '广东',
            sampleSize: 630,
            capturedAt: '2026-07-31T00:00:00Z',
            methodology: '官方目录整理',
            updatedAt: '2026-07-31T00:00:00Z',
          }],
          meta: { requestId: 'observation-insights' },
        },
      })
    })
    await page.route('**/api/v1/topics', async (route) => {
      await route.fulfill({ json: { data: [], meta: { requestId: 'observation-topics' } } })
    })

    await page.goto('/observe')
    await expect(page.getByRole('heading', { name: '广东招生计划选科观察' })).toBeVisible()
    await expect(page.getByRole('link', { name: '物理+化学 630 个' })).toBeVisible()
    await expect(page.getByRole('link', { name: /来源：广东考试院/ })).toBeVisible()

    await page.unroute('**/api/v1/insights')
    await page.route('**/api/v1/insights', async (route) => {
      await route.fulfill({ status: 503, json: { error: { code: 'unavailable', message: '暂时不可用', requestId: 'insights-error' } } })
    })
    await page.goto('/observe')
    await expect(page.getByRole('heading', { name: '观察数据暂时无法加载' })).toBeVisible()
    await expect(page.getByRole('button', { name: '重新加载' })).toBeVisible()
  })
})
