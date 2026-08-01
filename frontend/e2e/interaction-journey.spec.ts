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

test('signed-in user can like, favorite, and comment on a post', async ({ page }) => {
  let liked = false
  let favorited = false
  let comments: Array<{
    id: number
    postId: number
    userId: number
    author: string
    role: string
    content: string
    createdAt: string
  }> = []

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

    if (url.pathname.endsWith('/me/profile')) {
      await route.fulfill({ json: { data: { user, choiceProfile: {}, stats: {}, posts: [], comments: [], favorites: [], following: [], followers: [], bio: '', viewerFollowing: false }, meta: { requestId: 'e2e-profile' } } })
      return
    }
    if (url.pathname.endsWith('/notifications')) {
      await route.fulfill({ json: { data: [], meta: { requestId: 'e2e-notifications' } } })
      return
    }
    if (url.pathname.endsWith('/posts/42') && request.method() === 'GET') {
      await route.fulfill({
        json: {
          data: {
            post: {
              id: 42,
              userId: 9,
              authorName: '另一位同学',
              authorRole: 'student',
              title: '物化生如何选择专业',
              content: '想了解广东地区的专业覆盖情况。',
              imageUrls: [],
              tags: ['广东'],
              track: 'physics',
              electives: ['chemistry', 'biology'],
              category: 'question',
              grade: '高一',
              province: '广东',
              likesCount: 2,
            commentsCount: comments.length,
            favoritesCount: favorited ? 2 : 1,
            viewerLiked: liked,
            viewerFavorited: favorited,
              viewerFollowing: false,
              createdAt: '2026-07-31T00:00:00Z',
              updatedAt: '2026-07-31T00:00:00Z',
            },
            comments,
          },
          meta: { requestId: 'e2e-post-detail' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/like')) {
      liked = true
      await route.fulfill({ json: { data: { active: true, count: 3 }, meta: { requestId: 'e2e-like' } } })
      return
    }
    if (url.pathname.endsWith('/favorite')) {
      favorited = true
      await route.fulfill({ json: { data: { active: true, count: 2 }, meta: { requestId: 'e2e-favorite' } } })
      return
    }
    if (url.pathname.endsWith('/comments')) {
      comments = [{
        id: 100,
        postId: 42,
        userId: user.id,
        author: user.nickname,
        role: user.role,
        content: '这条评论来自公测互动 smoke。',
        createdAt: '2026-07-31T00:01:00Z',
      }]
      await route.fulfill({
        json: {
          data: comments[0],
          meta: { requestId: 'e2e-comment' },
        },
      })
      return
    }
    await route.fulfill({ json: { data: [], meta: { requestId: 'e2e-default' } } })
  })

  await page.goto('/posts/42')
  await expect(page.getByRole('heading', { name: '物化生如何选择专业' })).toBeVisible()

  await page.getByRole('button', { name: /2/ }).first().click()
  await expect(page.getByRole('button', { name: /3/ }).first()).toBeVisible()

  await page.getByRole('button', { name: '收藏' }).click()
  await expect(page.getByRole('button', { name: '已收藏' })).toBeVisible()

  await page.getByPlaceholder('写下你的看法，帮助更多正在选科的人').fill('这条评论来自公测互动 smoke。')
  await page.getByRole('button', { name: '发表评论' }).click()
  await expect(page.getByText('这条评论来自公测互动 smoke。')).toBeVisible()
})
