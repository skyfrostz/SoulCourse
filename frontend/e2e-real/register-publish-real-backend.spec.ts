import { expect, test } from '@playwright/test'

test('real backend user can register, publish, sign out, and sign back in', async ({ page }) => {
  const stamp = Date.now()
  const email = `real-e2e-${stamp}@example.com`
  const nickname = `真实联调${stamp % 100000}`
  const password = 'password123'
  const title = `真实后端发布 ${stamp}`

  await page.goto('/')
  await page.getByRole('button', { name: '登录 / 注册' }).click()
  await expect(page.getByRole('dialog', { name: '登录账号' })).toBeVisible()

  await page.getByRole('button', { name: '注册', exact: true }).click()
  await expect(page.getByRole('dialog', { name: '注册账号' })).toBeVisible()
  await expect(page.getByLabel('省份')).toHaveValue('广东')

  await page.getByPlaceholder('name@example.com').fill(email)
  await page.getByLabel('密码', { exact: true }).fill(password)
  await page.getByLabel('确认密码').fill(password)
  await page.getByPlaceholder('社区中显示的名字').fill(nickname)
  await page.getByRole('button', { name: '获取验证码' }).click()

  const feedback = page.locator('#auth-code-feedback')
  await expect(feedback).toContainText('本地调试验证码')
  const feedbackText = await feedback.textContent()
  const code = feedbackText?.match(/本地调试验证码：(\d{6})/)?.[1]
  expect(code, `debug code should be visible in local backend feedback: ${feedbackText}`).toBeTruthy()

  await page.getByPlaceholder('6 位验证码').fill(code!)
  await page.getByRole('button', { name: '创建账号' }).click()
  await expect(page.getByRole('button', { name: /个人中心/ })).toBeVisible()

  await page.getByRole('button', { name: '发帖' }).click()
  await expect(page.getByRole('heading', { name: '发布内容' })).toBeVisible()
  await page.getByLabel('标题').fill(title)
  await page.getByLabel('正文').fill('这是一条由 Playwright 连接真实 Go 后端创建的公测联调帖子。')
  await page.getByRole('button', { name: '发布', exact: true }).click()

  await expect(page).toHaveURL(/\/posts\/\d+$/)
  await expect(page.getByRole('heading', { name: title })).toBeVisible()
  await expect(page.getByRole('link', { name: new RegExp(nickname) })).toBeVisible()

  const publishedPostURL = page.url()

  await page.getByRole('button', { name: /个人中心/ }).click()
  await page.getByRole('button', { name: '退出登录' }).click()
  await expect(page.getByRole('button', { name: '登录 / 注册' })).toBeVisible()

  await page.getByRole('button', { name: '登录 / 注册' }).click()
  await expect(page.getByRole('dialog', { name: '登录账号' })).toBeVisible()
  await page.getByPlaceholder('name@example.com').fill(email)
  await page.getByLabel('密码', { exact: true }).fill(password)
  await page.getByRole('button', { name: '登录并继续' }).click()

  await expect(page.getByRole('button', { name: /个人中心/ })).toBeVisible()
  await page.goto(publishedPostURL)
  await expect(page.getByRole('heading', { name: title })).toBeVisible()
  await expect(page.getByRole('link', { name: new RegExp(nickname) })).toBeVisible()

  await page.getByRole('button', { name: /^0$/ }).click()
  await expect(page.getByRole('button', { name: /^1$/ })).toBeVisible()
  await page.getByRole('button', { name: '收藏' }).click()
  await expect(page.getByRole('button', { name: '已收藏' })).toBeVisible()

  const comment = `真实后端评论 ${stamp}`
  await page.getByPlaceholder('写下你的看法，帮助更多正在选科的人').fill(comment)
  await page.getByRole('button', { name: '发表评论' }).click()
  await expect(page.getByText(comment)).toBeVisible()

  await page.goto('/settings')
  await expect(page.getByRole('heading', { name: '账号安全' })).toBeVisible()
  await expect(page.getByText('当前设备')).toBeVisible()
  await page.getByPlaceholder('注销账号').fill('注销账号')
  await page.getByPlaceholder('请输入当前密码').fill(password)
  await page.getByRole('button', { name: '确认注销账号' }).click()
  const navLoginButton = page.getByRole('banner').getByRole('button', { name: '登录 / 注册' })
  await expect(navLoginButton).toBeVisible()

  await navLoginButton.click()
  await page.getByPlaceholder('name@example.com').fill(email)
  await page.getByLabel('密码', { exact: true }).fill(password)
  await page.getByRole('button', { name: '登录并继续' }).click()
  await expect(page.getByText('邮箱或密码不正确')).toBeVisible()
})
