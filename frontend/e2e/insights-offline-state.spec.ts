import { expect, test } from '@playwright/test'
import type { BrowserContext, Page } from '@playwright/test'

async function expectOffline(page: Page, context: BrowserContext) {
  await context.setOffline(true)
  await page.evaluate(() => window.dispatchEvent(new Event('offline')))
  await expect(page.getByRole('heading', { name: '当前网络不可用' })).toBeVisible()
  await context.setOffline(false)
}

test('insight, observation, and policy routes distinguish offline failures from empty data', async ({ page, context }) => {
  await page.route('**/api/v1/insights**', (route) => route.abort('failed'))
  await page.route('**/api/v1/provinces**', (route) => route.abort('failed'))
  await page.route('**/api/v1/policies**', (route) => route.abort('failed'))
  await page.route('**/api/v1/topics**', (route) => route.abort('failed'))
  await page.route('**/api/v1/posts**', (route) => route.abort('failed'))

  await page.goto('/insights')
  await expectOffline(page, context)

  await page.goto('/insights/1')
  await expectOffline(page, context)

  await page.goto('/observe')
  await expectOffline(page, context)

  await page.goto('/knowledge/广东/docs/missing')
  await expectOffline(page, context)
})
