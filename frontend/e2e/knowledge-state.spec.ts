import { expect, test } from '@playwright/test'

test('knowledge base shows a retryable error when verified data fails to load', async ({ page }) => {
  let provinceAttempts = 0
  await page.route('**/api/v1/provinces**', async (route) => {
    provinceAttempts += 1
    if (provinceAttempts === 1) {
      await route.fulfill({
        status: 503,
        json: { error: { code: 'service_unavailable', message: 'temporary provinces failure', requestId: 'e2e-provinces-error' } },
      })
      return
    }
    await route.fulfill({
      json: {
        data: {
          provinces: [{
            province: '广东',
            coverageStatus: 'verified',
            recordsCount: 2,
            dataYear: 2026,
            capturedAt: '2026-07-31T00:00:00Z',
            methodology: '官方来源结构化整理',
          }],
        },
        meta: { requestId: 'e2e-provinces-ok' },
      },
    })
  })
  await page.route('**/api/v1/policies**', (route) => route.fulfill({ json: { data: { policies: [] }, meta: { requestId: 'e2e-policies' } } }))

  await page.goto('/knowledge')

  await expect(page.getByRole('heading', { name: '政策资料加载失败' })).toBeVisible()
  await page.getByRole('button', { name: '重试' }).click()
  await expect(page.getByRole('link', { name: /广东/ }).first()).toBeVisible()
})

test('province detail separates API failure from missing province records', async ({ page }) => {
  let provinceAttempts = 0
  await page.route('**/api/v1/provinces**', async (route) => {
    provinceAttempts += 1
    if (provinceAttempts === 1) {
      await route.fulfill({
        status: 503,
        json: { error: { code: 'service_unavailable', message: 'temporary province failure', requestId: 'e2e-province-error' } },
      })
      return
    }
    await route.fulfill({
      json: {
        data: {
          provinces: [{
            province: '广东',
            coverageStatus: 'verified',
            recordsCount: 2,
            dataYear: 2026,
            capturedAt: '2026-07-31T00:00:00Z',
            methodology: '官方来源结构化整理',
          }],
        },
        meta: { requestId: 'e2e-province-ok' },
      },
    })
  })
  await page.route('**/api/v1/policies**', (route) => route.fulfill({ json: { data: { policies: [] }, meta: { requestId: 'e2e-province-policies' } } }))
  await page.route('**/api/v1/posts**', (route) => route.fulfill({ json: { data: [], meta: { requestId: 'e2e-province-posts' } } }))

  await page.goto('/knowledge/%E5%B9%BF%E4%B8%9C')

  await expect(page.getByRole('heading', { name: '省份资料加载失败' })).toBeVisible()
  await expect(page.getByText('没有找到该省份')).toBeHidden()

  await page.getByRole('button', { name: '重试' }).click()
  await expect(page.getByRole('heading', { name: '广东招生考试与选科文件' })).toBeVisible()
})
