import { expect, test } from '@playwright/test'
import { injectAxe } from 'axe-playwright'

const post = {
  id: 89,
  userId: 9,
  authorName: '多图同学',
  authorRole: 'student',
  title: '纵向长图与横图混排的帖子',
  content: '这是用于详情页验收的多图帖子。正文应该在桌面右栏独立滚动，在手机自然向下排列。',
  imageUrls: [
    'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" width="600" height="1200"><rect width="100%" height="100%" fill="%23202124"/><text x="50%" y="50%" fill="white" font-size="42" text-anchor="middle">Long image</text></svg>',
    'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="600"><rect width="100%" height="100%" fill="%230f766e"/><text x="50%" y="50%" fill="white" font-size="42" text-anchor="middle">Wide image</text></svg>',
    'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" width="800" height="800"><rect width="100%" height="100%" fill="%23b45309"/><text x="50%" y="50%" fill="white" font-size="42" text-anchor="middle">Square image</text></svg>',
  ],
  tags: ['验收'],
  track: 'physics',
  electives: ['chemistry', 'biology'],
  category: 'experience',
  grade: '高一',
  province: '广东',
  likesCount: 3,
  commentsCount: 0,
  favoritesCount: 1,
  viewerLiked: false,
  viewerFavorited: false,
  viewerFollowing: false,
  createdAt: '2026-07-31T00:00:00Z',
  updatedAt: '2026-07-31T00:00:00Z',
}

test('multi-image detail keeps a stable carousel across target widths', async ({ page }) => {
  await page.route('**/api/v1/posts/89', async (route) => route.fulfill({ json: { data: { post, comments: [] }, meta: { requestId: 'post-89' } } }))

  for (const width of [375, 768, 1024, 1280, 1440]) {
    await page.setViewportSize({ width, height: 900 })
    await page.goto('/posts/89')
    await expect(page.locator('.post-image-carousel')).toBeVisible()
    await expect(page.locator('.carousel-counter')).toHaveText('1 / 3')
    await injectAxe(page)
    const violations = await page.evaluate(async () => {
      const target = document.querySelector('.post-image-carousel')
      return (window as any).axe.run(target)
    })
    expect(violations.violations.filter((item: { impact?: string }) => item.impact === 'critical' || item.impact === 'serious')).toEqual([])
    await expect(page.locator('body')).toHaveJSProperty('scrollWidth', await page.locator('body').evaluate((node) => node.clientWidth))

    const stage = page.locator('.carousel-stage')
    await expect(stage).toHaveCSS('height', width < 1024 ? /.+/ : /.+/)
    await page.getByRole('button', { name: '下一张图片' }).click()
    await expect(page.locator('.carousel-counter')).toHaveText('2 / 3')
    await page.screenshot({ path: `/tmp/soulcourse-post-89-${width}.png`, fullPage: true })

    const mediaBox = await page.locator('.article-media-column').boundingBox()
    const contentBox = await page.locator('.article-main').boundingBox()
    expect(mediaBox).not.toBeNull()
    expect(contentBox).not.toBeNull()
    if (width >= 1024) {
      expect(contentBox!.x).toBeGreaterThan(mediaBox!.x + mediaBox!.width - 2)
    } else {
      expect(contentBox!.y).toBeGreaterThan(mediaBox!.y + mediaBox!.height - 2)
    }
  }
})
