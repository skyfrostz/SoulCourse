import { expect, test } from '@playwright/test'

const insights = [
  {
    id: 1,
    combination: '物理+化学',
    trend: '稳定上升',
    heat: 892,
    matchRate: 84,
    advice: '理工医类专业覆盖最广，适合目标尚不明确但想保留专业选择空间的考生。',
    details: '在广东和多数 3+1+2 省份中，物化组合仍是高覆盖率的主流选择。',
    metricType: '热度',
    unit: '热度值',
    province: '广东',
    dataYear: 2026,
    sourceName: '广东考试院',
    sourceUrl: 'https://example.test/insight-source',
    scope: '广东已复核招生目录',
    sampleSize: 1200,
    capturedAt: '2026-07-31T00:00:00Z',
    methodology: '基于官方目录和招生计划复算',
    updatedAt: '2026-07-31T00:00:00Z',
  },
  {
    id: 2,
    combination: '物理+地理',
    trend: '稳步变化',
    heat: 612,
    matchRate: 52,
    advice: '适合希望兼顾理工和地理相关专业的学生，但在医工方向覆盖较弱。',
    details: '该组合在部分专业中具备一定覆盖率，但不如物化稳定。',
    metricType: '热度',
    unit: '热度值',
    province: '广东',
    dataYear: 2026,
    sourceName: '广东考试院',
    sourceUrl: 'https://example.test/insight-source-geo',
    scope: '广东已复核招生目录',
    sampleSize: 980,
    capturedAt: '2026-07-31T00:00:00Z',
    methodology: '基于官方目录和招生计划复算',
    updatedAt: '2026-07-31T00:00:00Z',
  },
] as const

const provinces = {
  provinces: [
    {
      province: '广东',
      coverageStatus: 'verified',
      recordsCount: 2,
      dataYear: 2026,
      capturedAt: '2026-07-31T00:00:00Z',
      methodology: '官方来源结构化整理',
    },
    {
      province: '浙江',
      coverageStatus: 'unverified',
      recordsCount: 1,
      dataYear: 2026,
      capturedAt: '2026-07-31T00:00:00Z',
      methodology: '待复核来源整理',
    },
  ],
}

const policies = {
  policies: [
    {
      id: 'policy-gd',
      title: '广东招生政策摘要',
      type: '招生政策',
      scope: '广东',
      coverageStatus: 'verified',
      dataYear: 2026,
      capturedAt: '2026-07-31T00:00:00Z',
      source: { name: '广东考试院', url: 'https://example.test/gd-policy' },
      fileHash: 'sha256:gd-policy',
      methodology: '官方入口已收录，结构化整理。',
      summary: '广东已复核政策文件。',
      tags: ['广东', '政策'],
      url: 'https://example.test/gd-policy',
    },
  ],
}

const posts = [
  {
    id: 42,
    authorName: '广东学生',
    authorRole: 'student',
    title: '物化组合如何稳住专业覆盖',
    content: '想在广东兼顾医学和工科，物化组合是不是最稳？',
    imageUrls: [],
    tags: ['物化', '广东'],
    track: 'physics',
    electives: ['chemistry', 'biology'],
    category: 'question',
    grade: '高一',
    province: '广东',
    likesCount: 26,
    commentsCount: 6,
    favoritesCount: 3,
    viewerLiked: false,
    viewerFavorited: false,
    viewerFollowing: false,
    createdAt: '2026-07-31T00:12:00Z',
    updatedAt: '2026-07-31T00:12:00Z',
  },
]

test.describe('insights overview and detail', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/insights**', async (route) => {
      const url = new URL(route.request().url())
      if (url.pathname.endsWith('/insights/1')) {
        await route.fulfill({ json: { data: insights[0], meta: { requestId: 'insight-detail' } } })
        return
      }
      await route.fulfill({ json: { data: insights, meta: { requestId: 'insights-overview' } } })
    })
    await page.route('**/api/v1/provinces**', async (route) => {
      await route.fulfill({ json: { data: provinces, meta: { requestId: 'insights-provinces' } } })
    })
    await page.route('**/api/v1/policies**', async (route) => {
      await route.fulfill({ json: { data: policies, meta: { requestId: 'insights-policies' } } })
    })
    await page.route('**/api/v1/posts**', async (route) => {
      await route.fulfill({ json: { data: posts, meta: { requestId: 'insight-posts' } } })
    })
  })

  test('shows verified insight data and opens the insight detail page', async ({ page }) => {
    await page.goto('/insights')

    await expect(page.getByRole('heading', { name: '官方选科要求数据中心' })).toBeVisible()
    await expect(page.getByRole('link', { name: /物理\+化学/ })).toBeVisible()
    await expect(page.getByText('已纳入复核范围')).toBeVisible()
    await expect(page.getByText('暂无官方公开数据')).toBeVisible()

    await page.getByRole('link', { name: /物理\+化学/ }).click()

    await expect(page).toHaveURL(/\/insights\/1$/)
    await expect(page.getByRole('heading', { name: '物理+化学' })).toBeVisible()
    await expect(page.getByText('来源：广东考试院')).toBeVisible()
    await expect(page.locator('.topic-feed-panel .post-hit-area').first()).toBeVisible()
  })

  test('keeps insight overview empty-state text truthful when no records exist', async ({ page }) => {
    await page.unroute('**/api/v1/insights**')
    await page.route('**/api/v1/insights**', async (route) => {
      await route.fulfill({ json: { data: [], meta: { requestId: 'insights-empty' } } })
    })

    await page.goto('/insights')

    await expect(page.locator('.insight-feature-card')).toHaveCount(0)
    await expect(page.getByRole('heading', { name: '省级官方来源与公开情况' })).toBeVisible()
  })
})
