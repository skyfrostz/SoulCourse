import { expect, test } from '@playwright/test'
import type { BrowserContext, Page } from '@playwright/test'

async function enterOffline(page: Page, context: BrowserContext) {
  await context.setOffline(true)
  await page.evaluate(() => window.dispatchEvent(new Event('offline')))
  await expect(page.getByRole('heading', { name: '当前网络不可用' })).toBeVisible()
  await context.setOffline(false)
}

test('real-data routes distinguish browser offline from missing verified data', async ({ page, context }) => {
  await page.route('**/api/v1/provinces**', (route) => route.abort('failed'))
  await page.route('**/api/v1/policies**', (route) => route.abort('failed'))
  await page.route('**/api/v1/posts**', (route) => route.abort('failed'))

  await page.goto('/knowledge')
  await enterOffline(page, context)

  await page.goto('/knowledge/广东')
  await enterOffline(page, context)

  await page.unroute('**/api/v1/provinces**')
  await page.unroute('**/api/v1/policies**')
  await page.route('**/api/v1/requirements**', (route) => route.abort('failed'))
  await page.goto('/requirements')
  await enterOffline(page, context)
  await expect(page.getByRole('button', { name: '重试' })).toBeVisible()

  await page.goto('/requirements/临床医学')
  await context.setOffline(true)
  await page.evaluate(() => window.dispatchEvent(new Event('offline')))
  await expect(page.getByText('当前网络不可用，官方要求尚未同步；社区讨论仍可查看。')).toBeVisible()
  await context.setOffline(false)
})
