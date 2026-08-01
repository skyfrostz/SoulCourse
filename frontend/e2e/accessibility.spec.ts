import { expect, test } from '@playwright/test'
import { injectAxe } from 'axe-playwright'

async function audit(page: Parameters<typeof injectAxe>[0], label: string) {
  await page.waitForTimeout(400)
  await injectAxe(page)
  const violations = await page.evaluate(async () => {
    const result = await (window as any).axe.run(document)
    return result.violations
      .filter((violation: { impact?: string }) => violation.impact === 'critical' || violation.impact === 'serious')
      .map((violation: { id: string; nodes: Array<{ target: string[] }> }) => ({
        id: violation.id,
        targets: violation.nodes.map((node) => node.target),
      }))
  })
  expect(violations, `${label} critical/serious accessibility violations`).toEqual([])
}

test.describe('critical public pages accessibility', () => {
  test('404 recovery has no critical or serious axe violations', async ({ page }) => {
    await page.goto('/a11y-route-does-not-exist')
    await audit(page, '404')
  })

  test('topic empty state has no critical or serious axe violations', async ({ page }) => {
    await page.route('**/api/v1/topics', async (route) => {
      await route.fulfill({ json: { data: [], meta: { requestId: 'a11y-topics' } } })
    })
    await page.goto('/topics')
    await expect(page.getByRole('heading', { name: '暂时没有已发布话题' })).toBeVisible()
    await audit(page, 'topics')
  })

  test('policy empty state has no critical or serious axe violations', async ({ page }) => {
    await page.route('**/api/v1/policies**', async (route) => {
      await route.fulfill({ json: { data: { policies: [] }, meta: { requestId: 'a11y-policies' } } })
    })
    await page.goto('/knowledge/广东/docs/missing')
    await expect(page.getByRole('heading', { name: '没有找到已复核政策记录' })).toBeVisible()
    await audit(page, 'policy-document')
  })
})
