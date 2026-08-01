import { expect, test } from '@playwright/test'

const topic = {
  id: 7,
  slug: 'physics-chemistry',
  topicTag: '物理方向',
  title: '物理方向怎么选',
  summary: '围绕物理方向的真实经验讨论。',
  viewsCount: 1280,
  postsCount: 1,
  createdAt: '2026-07-31T00:00:00Z',
}

test.describe('topics and policy document public-beta states', () => {
  test('opens a real topic and shows its related discussion', async ({ page }) => {
    await page.route('**/api/v1/topics', async (route) => {
      await route.fulfill({ json: { data: [topic], meta: { requestId: 'topics-overview' } } })
    })
    await page.route('**/api/v1/topics/physics-chemistry', async (route) => {
      await route.fulfill({
        json: {
          data: {
            topic,
            posts: [{
              id: 91,
              authorName: '广东学生',
              authorRole: 'student',
              title: '物理方向的专业覆盖怎么查',
              content: '想先核对官方要求。',
              imageUrls: [],
              tags: ['物理方向'],
              track: 'physics',
              electives: ['chemistry'],
              category: 'question',
              grade: '高一',
              province: '广东',
              likesCount: 2,
              commentsCount: 1,
              favoritesCount: 0,
              viewerLiked: false,
              viewerFavorited: false,
              viewerFollowing: false,
              createdAt: '2026-07-31T00:00:00Z',
              updatedAt: '2026-07-31T00:00:00Z',
            }],
          },
          meta: { requestId: 'topic-detail' },
        },
      })
    })

    await page.goto('/topics')
    await expect(page.getByRole('heading', { name: '热门话题广场' })).toBeVisible()
    await expect(page.getByRole('link', { name: /物理方向怎么选/ })).toBeVisible()
    await page.getByRole('link', { name: /物理方向怎么选/ }).click()
    await expect(page).toHaveURL(/\/topics\/physics-chemistry$/)
    await expect(page.getByRole('heading', { name: '# 物理方向怎么选' })).toBeVisible()
    await expect(page.locator('.topic-feed-panel .post-hit-area').getByText('物理方向的专业覆盖怎么查')).toBeVisible()
  })

  test('shows truthful empty state when no topics are published', async ({ page }) => {
    await page.route('**/api/v1/topics', async (route) => {
      await route.fulfill({ json: { data: [], meta: { requestId: 'topics-empty' } } })
    })
    await page.goto('/topics')
    await expect(page.getByRole('heading', { name: '暂时没有已发布话题' })).toBeVisible()
    await expect(page.getByText('不会用模拟话题')).not.toBeVisible()
  })

  test('shows the verified policy document source record', async ({ page }) => {
    await page.route('**/api/v1/policies**', async (route) => {
      await route.fulfill({
        json: {
          data: {
            policies: [{
              id: 'policy-gd-1',
              title: '广东招生选科文件',
              type: '招生政策',
              scope: '广东',
              coverageStatus: 'verified',
              dataYear: 2026,
              capturedAt: '2026-07-31T00:00:00Z',
              source: { name: '广东考试院', url: 'https://example.test/gd-source' },
              fileHash: 'sha256:gd-policy',
              methodology: '官方入口已收录，结构化整理。',
              summary: '广东已复核政策文件。',
              tags: ['广东', '选科'],
              url: 'https://example.test/gd-policy',
            }],
          },
          meta: { requestId: 'policy-document' },
        },
      })
    })
    await page.goto('/knowledge/广东/docs/policy-gd-1')
    await expect(page.getByRole('heading', { name: '广东招生选科文件' })).toBeVisible()
    await expect(page.getByText('广东考试院')).toBeVisible()
    await expect(page.getByText('sha256:gd-policy')).toBeVisible()
    await expect(page.getByRole('link', { name: '官方来源' })).toHaveAttribute('href', 'https://example.test/gd-source')
  })
})
