import { expect, test } from '@playwright/test'

test.describe('not found recovery', () => {
  test('shows a usable 404 page on desktop and mobile', async ({ page }) => {
    await page.goto('/this-route-should-not-exist')

    await expect(page.getByRole('heading', { name: '页面不存在' })).toBeVisible()
    await expect(page.getByRole('link', { name: '回到首页' })).toBeVisible()
    await expect(page.getByRole('button', { name: '返回上一页' })).toBeVisible()
  })

  test.describe('mobile viewport', () => {
    test.use({ viewport: { width: 390, height: 844 } })

    test('keeps the 404 recovery state usable', async ({ page }) => {
      await page.goto('/still-not-found')

      await expect(page.getByRole('heading', { name: '页面不存在' })).toBeVisible()
      await expect(page.getByRole('link', { name: '回到首页' })).toBeVisible()
      await expect(page.getByRole('button', { name: '返回上一页' })).toBeVisible()
    })
  })
})
