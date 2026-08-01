import { expect, test } from '@playwright/test'

test('knowledge pages surface verified coverage and province detail content', async ({ page }) => {
  await page.route('**/api/v1/provinces**', async (route) => {
    await route.fulfill({
      json: {
        data: {
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
        },
        meta: { requestId: 'knowledge-provinces' },
      },
    })
  })
  await page.route('**/api/v1/policies**', async (route) => {
    await route.fulfill({
      json: {
        data: {
          policies: [
            {
              id: 'policy-gd-1',
              title: '广东招生选科文件',
              type: '招生政策',
              scope: '广东',
              coverageStatus: 'verified',
              dataYear: 2026,
              capturedAt: '2026-07-31T00:00:00Z',
              source: { name: '广东考试院', url: 'https://example.test/gd-source' },
              fileHash: 'sha256:gd',
              methodology: '官方入口已收录，结构化整理。',
              summary: '广东已复核政策文件。',
              tags: ['广东', '选科'],
              url: 'https://example.test/gd-policy',
            },
          ],
        },
        meta: { requestId: 'knowledge-policies' },
      },
    })
  })
  await page.route('**/api/v1/posts**', async (route) => {
    await route.fulfill({ json: { data: [], meta: { requestId: 'knowledge-posts' } } })
  })

  await page.goto('/knowledge')

  await expect(page.getByRole('heading', { name: '全国招生考试与选科知识库' })).toBeVisible()
  await expect(page.getByRole('link', { name: /官方已复核.*广东.*已复核数据.*2026/ })).toBeVisible()
  await expect(page.getByText('已发布结构化数据')).toBeVisible()
  await expect(page.getByText('不展示模拟结论')).toBeVisible()

  await page.goto('/knowledge/%E5%B9%BF%E4%B8%9C')

  await expect(page.getByRole('heading', { name: '广东招生考试与选科文件' })).toBeVisible()
  await expect(page.getByText('可查看已复核文件')).toBeVisible()
  await expect(page.getByRole('link', { name: '查看网页化全文' })).toBeVisible()
  await expect(page.getByRole('link', { name: '官方/下载入口' })).toBeVisible()
})
