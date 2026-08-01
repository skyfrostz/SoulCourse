import { expect, test } from '@playwright/test'

const user = {
  id: 7,
  publicId: 'u_7',
  email: 'student@example.com',
  nickname: '广东学生',
  role: 'student',
  province: '广东',
  grade: '高一',
  createdAt: '2026-07-31T00:00:00Z',
}

test('signed-in user can open a conversation and send a message', async ({ page }) => {
  let sentMessages = [
    {
      id: 301,
      senderId: 9,
      senderName: '规划老师',
      recipientId: user.id,
      recipientName: user.nickname,
      content: '你好，可以先说说你的目标专业。',
      createdAt: '2026-07-31T00:00:00Z',
    },
  ]

  await page.addInitScript(({ session }) => {
    localStorage.setItem('scf_auth_session', JSON.stringify(session))
  }, {
    session: {
      user,
      token: 'e2e-session-token',
      expiresAt: '2099-01-01T00:00:00Z',
    },
  })

  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (url.pathname.endsWith('/messages') && request.method() === 'GET') {
      await route.fulfill({
        json: {
          data: [{
            user: {
              id: 9,
              publicId: 'u_9',
              email: 'planner@example.com',
              nickname: '规划老师',
              role: 'counselor',
              province: '广东',
              grade: '高中',
              createdAt: '2026-07-30T00:00:00Z',
            },
            lastMessage: sentMessages.at(-1)?.content ?? '',
            lastMessageAt: sentMessages.at(-1)?.createdAt ?? '2026-07-30T00:00:00Z',
            unreadCount: 0,
          }],
          meta: { requestId: 'e2e-conversations' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/messages/规划老师') && request.method() === 'GET') {
      await route.fulfill({ json: { data: sentMessages, meta: { requestId: 'e2e-thread' } } })
      return
    }
    if (url.pathname.endsWith('/messages') && request.method() === 'POST') {
      sentMessages = [...sentMessages, {
        id: 302,
        senderId: user.id,
        senderName: user.nickname,
        recipientId: 9,
        recipientName: '规划老师',
        content: '我想了解物化生在广东的专业覆盖。',
        createdAt: '2026-07-31T00:01:00Z',
      }]
      await route.fulfill({ json: { data: sentMessages.at(-1), meta: { requestId: 'e2e-send-message' } } })
      return
    }
    await route.fulfill({ json: { data: [], meta: { requestId: 'e2e-default' } } })
  })

  await page.goto('/messages?to=%E8%A7%84%E5%88%92%E8%80%81%E5%B8%88')
  await expect(page.getByRole('heading', { name: '私信' })).toBeVisible()
  await expect(page.getByText('你好，可以先说说你的目标专业。')).toBeVisible()

  await page.getByPlaceholder('发消息给 规划老师').fill('我想了解物化生在广东的专业覆盖。')
  await page.getByRole('button', { name: '发送消息' }).click()
  await expect(page.getByText('我想了解物化生在广东的专业覆盖。')).toBeVisible()
})

test('message send failure keeps the draft and shows retry feedback', async ({ page }) => {
  await page.addInitScript(({ session }) => {
    localStorage.setItem('scf_auth_session', JSON.stringify(session))
  }, {
    session: {
      user,
      token: 'e2e-session-token',
      expiresAt: '2099-01-01T00:00:00Z',
    },
  })

  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (url.pathname.endsWith('/messages') && request.method() === 'GET') {
      await route.fulfill({
        json: {
          data: [{
            user: {
              id: 9,
              publicId: 'u_9',
              email: 'planner@example.com',
              nickname: '规划老师',
              role: 'counselor',
              province: '广东',
              grade: '高中',
              createdAt: '2026-07-30T00:00:00Z',
            },
            lastMessage: '你好，可以先说说你的目标专业。',
            lastMessageAt: '2026-07-31T00:00:00Z',
            unreadCount: 0,
          }],
          meta: { requestId: 'e2e-conversations' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/messages/规划老师') && request.method() === 'GET') {
      await route.fulfill({ json: { data: [], meta: { requestId: 'e2e-thread-empty' } } })
      return
    }
    if (url.pathname.endsWith('/messages') && request.method() === 'POST') {
      await route.fulfill({
        status: 503,
        json: { error: { code: 'service_unavailable', message: 'message send failed', requestId: 'e2e-message-send-failed' } },
      })
      return
    }
    await route.fulfill({ json: { data: [], meta: { requestId: 'e2e-default' } } })
  })

  await page.goto('/messages?to=%E8%A7%84%E5%88%92%E8%80%81%E5%B8%88')
  const composer = page.getByPlaceholder('发消息给 规划老师')
  await composer.fill('这条失败后不能丢。')
  await page.getByRole('button', { name: '发送消息' }).click()

  await expect(page.getByRole('alert')).toContainText('草稿已保留')
  await expect(composer).toHaveValue('这条失败后不能丢。')
})
