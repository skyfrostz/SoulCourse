import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'

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

const choiceProfile = {
  realName: '', city: '', schoolType: '', gradeRank: '', mbti: '', targetMajors: '', targetCities: '',
  subjectStability: '', physicsScore: '', historyScore: '', chemistryScore: '', biologyScore: '', politicsScore: '', geographyScore: '',
  preferredTrack: 'physics', preferredSubjects: ['chemistry', 'biology'], learningStyle: '', pressureTolerance: '', recommendationFocus: '',
}

const widths = [375, 390, 768, 1024, 1280, 1440]
const height = 900

test('critical public beta routes do not create horizontal overflow across target widths', async ({ page }) => {
  await seedSignedInSession(page)
  await mockResponsiveApi(page)

  const routes = [
    { path: '/', readyText: '物化生如何选择专业' },
    { path: '/posts/42', readyText: '物化生如何选择专业' },
    { path: '/users/%E5%B9%BF%E4%B8%9C%E5%AD%A6%E7%94%9F', readyText: '广东学生', readySelector: '.user-profile-hero h1' },
    { path: '/following', readyText: '我的关注', readySelector: '.following-hero h1' },
    { path: '/settings', readyText: '个人信息与选科画像', readySelector: '.settings-hero h1' },
    { path: '/notifications', readySelector: '.notifications-page' },
    { path: '/messages?to=%E8%A7%84%E5%88%92%E8%80%81%E5%B8%88', readyText: '发消息给 规划老师' },
    { path: '/requirements', readyText: '像刷笔记一样查专业选科' },
    { path: '/requirements/%E4%B8%B4%E5%BA%8A%E5%8C%BB%E5%AD%A6', readyText: '临床医学讨论论坛' },
    { path: '/knowledge', readyText: '全国招生考试与选科知识库' },
    { path: '/topics', readyText: '热门话题广场' },
    { path: '/topics/responsive-topic', readyText: '# 广东选科讨论' },
    { path: '/insights', readyText: '官方选科要求数据中心' },
    { path: '/insights/1', readyText: '物理+化学' },
    { path: '/advice', readyText: '同一帖子库里的选科经验' },
    { path: '/advice/42', readyText: '物化生如何选择专业' },
    { path: '/observe', readyText: '暂时没有已发布观察数据' },
    { path: '/knowledge/%E5%B9%BF%E4%B8%9C/docs/policy-gd', readyText: '广东 2026 选科政策' },
    { path: '/admin', readyText: '选科π管理后台' },
  ]

  for (const width of widths) {
    await page.setViewportSize({ width, height })
    for (const route of routes) {
      await page.goto(route.path)
      if ('readySelector' in route && route.readySelector) {
        if ('readyText' in route && route.readyText) {
          await expect(page.locator(route.readySelector)).toHaveText(route.readyText)
        } else {
          await expect(page.locator(route.readySelector)).toBeVisible()
        }
      } else if (route.path.startsWith('/messages')) {
        await expect(page.getByPlaceholder(route.readyText)).toBeVisible()
      } else {
        await expect(page.getByText(route.readyText).first()).toBeVisible()
      }
      await expectNoHorizontalOverflow(page, `${route.path} @ ${width}px`)
    }
  }
})

async function seedSignedInSession(page: Page) {
  await page.addInitScript(({ session }) => {
    localStorage.setItem('scf_auth_session', JSON.stringify(session))
  }, {
    session: {
      user,
      token: 'e2e-session-token',
      expiresAt: '2099-01-01T00:00:00Z',
    },
  })
}

async function mockResponsiveApi(page: Page) {
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (url.pathname.endsWith('/me/profile')) {
      await route.fulfill({
        json: {
          data: {
            user,
            bio: '',
            choiceProfile,
            stats: { posts: 0, comments: 0, following: 0, followers: 0, favorites: 0, engagement: 0 },
            posts: [],
            comments: [],
            favorites: [],
            following: [],
            followers: [],
            viewerFollowing: false,
          },
          meta: { requestId: 'responsive-profile' },
        },
      })
      return
    }
    if (decodeURIComponent(url.pathname).endsWith('/profiles/广东学生')) {
      await route.fulfill({
        json: {
          data: {
            user,
            bio: '关注广东新高考选科。',
            choiceProfile,
            stats: { posts: 0, comments: 0, following: 0, followers: 0, favorites: 0, engagement: 0 },
            posts: [],
            comments: [],
            favorites: [],
            following: [],
            followers: [],
            viewerFollowing: false,
          },
          meta: { requestId: 'responsive-public-profile' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/notifications')) {
      await route.fulfill({ json: { data: [], meta: { requestId: 'responsive-notifications' } } })
      return
    }
    if (url.pathname.endsWith('/messages') && request.method() === 'GET') {
      await route.fulfill({
        json: {
          data: [{
            user: { ...user, id: 9, nickname: '规划老师', role: 'counselor' },
            lastMessage: '你好，可以先说说你的目标专业。',
            lastMessageAt: '2026-07-31T00:00:00Z',
            unreadCount: 0,
          }],
          meta: { requestId: 'responsive-conversations' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/messages/规划老师')) {
      await route.fulfill({
        json: {
          data: [{
            id: 301,
            senderId: 9,
            senderName: '规划老师',
            recipientId: user.id,
            recipientName: user.nickname,
            content: '你好，可以先说说你的目标专业。',
            createdAt: '2026-07-31T00:00:00Z',
          }],
          meta: { requestId: 'responsive-thread' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/posts/42')) {
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
              commentsCount: 0,
              favoritesCount: 1,
              viewerLiked: false,
              viewerFavorited: false,
              viewerFollowing: false,
              createdAt: '2026-07-31T00:00:00Z',
              updatedAt: '2026-07-31T00:00:00Z',
            },
            comments: [],
          },
          meta: { requestId: 'responsive-post-detail' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/posts')) {
      await route.fulfill({
        json: {
          data: [{
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
            commentsCount: 0,
            favoritesCount: 1,
            viewerLiked: false,
            viewerFavorited: false,
            viewerFollowing: false,
            createdAt: '2026-07-31T00:00:00Z',
            updatedAt: '2026-07-31T00:00:00Z',
          }],
          meta: { requestId: 'responsive-posts', hasMore: false },
        },
      })
      return
    }
    if (url.pathname.endsWith('/requirements')) {
      await route.fulfill({
        json: {
          data: {
            requirements: [{
              id: 'req-gd-computer',
              title: '计算机类选科要求',
              type: '物理+化学强约束',
              scope: '广东',
              coverageStatus: 'verified',
              dataYear: 2026,
              capturedAt: '2026-07-31T00:00:00Z',
              source: { name: '广东省教育考试院', url: 'https://example.com' },
              fileHash: 'sha256:responsive',
              methodology: '官方来源结构化整理',
              summary: '计算机类专业通常要求物理和化学。',
              tags: ['广东', '计算机'],
              url: 'https://example.com',
            }],
          },
          meta: { requestId: 'responsive-requirements' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/provinces')) {
      await route.fulfill({
        json: {
          data: {
            provinces: [{
              province: '广东',
              coverageStatus: 'verified',
              recordsCount: 1,
              dataYear: 2026,
              capturedAt: '2026-07-31T00:00:00Z',
              methodology: '官方来源结构化整理',
            }],
          },
          meta: { requestId: 'responsive-provinces' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/policies')) {
      await route.fulfill({
        json: {
          data: {
            policies: [{
              id: 'policy-gd',
              title: '广东 2026 选科政策',
              type: '政策文件',
              scope: '广东',
              coverageStatus: 'verified',
              dataYear: 2026,
              capturedAt: '2026-07-31T00:00:00Z',
              source: { name: '广东省教育考试院', url: 'https://example.com' },
              fileHash: 'sha256:responsive-policy',
              methodology: '官方来源结构化整理',
              summary: '广东公测政策样例。',
              tags: ['广东'],
              url: 'https://example.com/policy.pdf',
            }],
          },
          meta: { requestId: 'responsive-policies' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/insights/1')) {
      await route.fulfill({
        json: {
          data: {
            id: 1,
            combination: '物理+化学',
            trend: '稳定',
            heat: 630,
            matchRate: 82,
            advice: '优先核对目标专业要求。',
            details: '广东已复核数据。',
            metricType: '计划数',
            unit: '个',
            province: '广东',
            dataYear: 2026,
            sourceName: '广东省教育考试院',
            sourceUrl: 'https://example.com',
            scope: '广东',
            sampleSize: 630,
            capturedAt: '2026-07-31T00:00:00Z',
            methodology: '官方目录整理',
            updatedAt: '2026-07-31T00:00:00Z',
          },
          meta: { requestId: 'responsive-insight-detail' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/topics/responsive-topic')) {
      await route.fulfill({
        json: {
          data: {
            topic: {
              id: 1,
              slug: 'responsive-topic',
              topicTag: '物理方向',
              title: '广东选科讨论',
              summary: '广东学生和家长的真实讨论。',
              viewsCount: 1200,
              postsCount: 0,
              createdAt: '2026-07-31T00:00:00Z',
            },
            posts: [],
          },
          meta: { requestId: 'responsive-topic-detail' },
        },
      })
      return
    }
    if (url.pathname.endsWith('/insights') || url.pathname.endsWith('/topics')) {
      await route.fulfill({ json: { data: [], meta: { requestId: 'responsive-public-empty' } } })
      return
    }
    await route.fulfill({ json: { data: {}, meta: { requestId: 'responsive-default' } } })
  })
}

async function expectNoHorizontalOverflow(page: Page, label: string) {
  const overflow = await page.evaluate(() => {
    function hasScrollableInlineAncestor(element: HTMLElement) {
      let current: HTMLElement | null = element.parentElement
      while (current && current !== document.body) {
        const style = window.getComputedStyle(current)
        const canScrollInline = ['auto', 'scroll'].includes(style.overflowX)
        if (canScrollInline && current.scrollWidth > current.clientWidth + 2) return true
        current = current.parentElement
      }
      return false
    }

    const documentElement = document.documentElement
    const body = document.body
    const viewportWidth = documentElement.clientWidth
    const offenders = Array.from(document.querySelectorAll<HTMLElement>('body *'))
      .map((element) => {
        const rect = element.getBoundingClientRect()
        const style = window.getComputedStyle(element)
        return {
          tag: element.tagName.toLowerCase(),
          className: String(element.className || ''),
          text: (element.textContent || '').trim().slice(0, 80),
          left: Math.floor(rect.left),
          right: Math.ceil(rect.right),
          display: style.display,
          visibility: style.visibility,
          insideInlineScroller: hasScrollableInlineAncestor(element),
        }
      })
      .filter((item) =>
        item.display !== 'none' &&
        item.visibility !== 'hidden' &&
        !item.insideInlineScroller &&
        (item.left < -2 || item.right > viewportWidth + 2),
      )
      .slice(0, 5)

    return {
      viewportWidth,
      documentScrollWidth: documentElement.scrollWidth,
      bodyScrollWidth: body.scrollWidth,
      offenders,
    }
  })

  expect(overflow.documentScrollWidth, `${label} document overflow: ${JSON.stringify(overflow)}`).toBeLessThanOrEqual(overflow.viewportWidth + 2)
  expect(overflow.bodyScrollWidth, `${label} body overflow: ${JSON.stringify(overflow)}`).toBeLessThanOrEqual(overflow.viewportWidth + 2)
  expect(overflow.offenders, `${label} overflowing elements`).toEqual([])
}
