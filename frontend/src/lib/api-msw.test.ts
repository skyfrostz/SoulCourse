import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../test/setup'
import { fetchNotifications, fetchPublishedRequirements } from './api'

describe('real data API with MSW', () => {
  it('reads the standard data envelope without page-level fallback data', async () => {
    server.use(http.get('/api/v1/requirements', () => HttpResponse.json({
      data: {
        requirements: [{
          id: 'req-1',
          title: '临床医学',
          type: '物理 + 化学',
          scope: '广东',
          coverageStatus: 'verified',
          dataYear: 2026,
          capturedAt: '2026-07-31T00:00:00Z',
          source: { name: '考试院', url: 'https://example.test/source' },
          fileHash: 'sha256:test',
          methodology: '逐校核验',
          summary: '官方目录记录',
          tags: ['医学'],
          url: 'https://example.test/record',
        }],
      },
      meta: { requestId: 'req-test' },
    })))

    await expect(fetchPublishedRequirements()).resolves.toMatchObject([{ title: '临床医学', coverageStatus: 'verified' }])
  })

  it('returns notification cursor metadata and sends pagination parameters', async () => {
    server.use(http.get('/api/v1/notifications', ({ request }) => {
      const url = new URL(request.url)
      expect(url.searchParams.get('limit')).toBe('30')
      expect(url.searchParams.get('cursor')).toBe('notification-page-2')
      return HttpResponse.json({
        data: [{ id: 2, type: 'system', title: '系统通知', summary: '内容', targetUrl: '/', createdAt: '2026-07-31T00:00:00Z' }],
        meta: { requestId: 'notification-test', nextCursor: 'notification-page-3', hasMore: true },
      })
    }))

    await expect(fetchNotifications({ limit: 30, cursor: 'notification-page-2' })).resolves.toMatchObject({
      items: [{ id: 2, title: '系统通知' }],
      nextCursor: 'notification-page-3',
      hasMore: true,
    })
  })
})
