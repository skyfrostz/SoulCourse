import axios from 'axios'
import { http, HttpResponse } from 'msw'
import { describe, expect, it, vi } from 'vitest'
import { server } from '../test/setup'
import { fetchNotifications, fetchPublishedRequirements, uploadImage } from './api'

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

  it('uploads to a presigned OSS URL without site credentials or CSRF headers', async () => {
    document.cookie = 'scf_csrf=site-csrf'
    server.use(
      http.post('/api/v1/uploads/images/presign', () => HttpResponse.json({
        data: {
          id: 'upload-1', assetKey: 'images/upload-1.png', uploadUrl: 'https://oss.example.test/upload-1',
          method: 'PUT', contentType: 'image/png', maxBytes: 1000, expiresAt: '2026-08-01T00:00:00Z',
        },
      })),
      http.post('/api/v1/uploads/images/upload-1/complete', () => HttpResponse.json({
        data: { id: 'upload-1', assetKey: 'images/upload-1.png', url: 'https://cdn.example.test/upload-1.png', contentType: 'image/png', sizeBytes: 3, width: 1, height: 1 },
      })),
    )
    const put = vi.spyOn(axios, 'put').mockResolvedValue({ data: {} } as never)

    await uploadImage(new File(['png'], 'upload.png', { type: 'image/png' }), { width: 1, height: 1 })

    expect(put).toHaveBeenCalledWith('https://oss.example.test/upload-1', expect.any(File), expect.objectContaining({
      withCredentials: false,
      headers: { 'Content-Type': 'image/png' },
    }))
    expect(put.mock.calls[0]?.[2]).not.toEqual(expect.objectContaining({ headers: expect.objectContaining({ 'X-CSRF-Token': 'site-csrf' }) }))
  })
})
