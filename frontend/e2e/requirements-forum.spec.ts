import { expect, test } from '@playwright/test'

const requirements = [
  {
    id: 'req-1',
    title: '临床医学',
    type: '物理 + 化学',
    scope: '广东',
    coverageStatus: 'verified',
    dataYear: 2026,
    capturedAt: '2026-07-31T00:00:00Z',
    source: { name: '广东考试院', url: 'https://example.test/gd-medical' },
    fileHash: 'sha256:gd-medical',
    methodology: '官方入口已收录，结构化整理。',
    summary: '临床医学已复核政策记录。',
    tags: ['广东', '医学', '临床'],
    url: 'https://example.test/gd-medical',
  },
  {
    id: 'req-2',
    title: '计算机科学与技术',
    type: '物理 + 化学',
    scope: '广东',
    coverageStatus: 'verified',
    dataYear: 2026,
    capturedAt: '2026-07-31T00:00:00Z',
    source: { name: '广东考试院', url: 'https://example.test/gd-computer' },
    fileHash: 'sha256:gd-computer',
    methodology: '官方入口已收录，结构化整理。',
    summary: '计算机科学与技术已复核政策记录。',
    tags: ['广东', '计算机', '人工智能'],
    url: 'https://example.test/gd-computer',
  },
] as const

const posts = [
  {
    id: 42,
    authorName: '广东学生',
    authorRole: 'student',
    title: '临床医学选科怎么配更稳',
    content: '我在广东想报临床医学，物化生之外还需要注意什么？',
    imageUrls: [],
    tags: ['临床医学', '广东'],
    track: 'physics',
    electives: ['chemistry', 'biology'],
    category: 'question',
    grade: '高一',
    province: '广东',
    likesCount: 16,
    commentsCount: 5,
    favoritesCount: 2,
    viewerLiked: false,
    viewerFavorited: false,
    viewerFollowing: false,
    createdAt: '2026-07-31T00:10:00Z',
    updatedAt: '2026-07-31T00:10:00Z',
  },
]

test.describe('requirements and major forum', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/requirements**', async (route) => {
      await route.fulfill({
        json: {
          data: { requirements },
          meta: { requestId: 'requirements-e2e' },
        },
      })
    })
    await page.route('**/api/v1/posts**', async (route) => {
      await route.fulfill({
        json: {
          data: posts,
          meta: { requestId: 'requirements-posts-e2e' },
        },
      })
    })
  })

  test('shows verified requirement cards and opens the matching forum', async ({ page }) => {
    await page.goto('/requirements')

    await expect(page.getByRole('heading', { name: '像刷笔记一样查专业选科' })).toBeVisible()
    await expect(page.locator('.requirement-note-card').filter({ hasText: '临床医学' })).toBeVisible()

    await page.locator('.requirement-note-card').filter({ hasText: '临床医学' }).click()

    await expect(page).toHaveURL(/\/requirements\/%E4%B8%B4%E5%BA%8A%E5%8C%BB%E5%AD%A6$/)
    await expect(page.getByRole('heading', { name: '临床医学讨论论坛' })).toBeVisible()
    await expect(page.getByText('选科要求摘要')).toBeVisible()
    await expect(page.locator('.major-forum-feed .post-hit-area').first()).toBeVisible()
  })

  test('filters to an empty state without fabricating results', async ({ page }) => {
    await page.goto('/requirements')

    await page.getByPlaceholder('搜索：临床医学、计算机、法学、师范...').fill('不存在')
    await expect(page.getByRole('heading', { name: '暂无已复核数据' })).toBeVisible()
    await expect(page.locator('.requirements-page .empty-state')).toContainText('当前筛选条件下没有已上架专业要求记录')
  })
})
