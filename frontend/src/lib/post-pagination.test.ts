import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../test/setup'
import { fetchFeedPage, postCollectionParams } from './api'
import type { FeedFilter } from '../types/forum'

const baseFilter: FeedFilter = {
  track: 'all',
  subjects: [],
  category: 'all',
  keyword: '',
  sort: 'latest',
}

describe('post collection request params', () => {
  it('keeps cursor pagination params and serializes subjects for the API contract', () => {
    expect(postCollectionParams({
      subjects: ['chemistry', 'biology'],
      sort: 'latest',
      limit: 20,
      cursor: '2026-07-31T00:00:00Z_42',
    })).toEqual({
      subjects: 'chemistry,biology',
      sort: 'latest',
      limit: 20,
      cursor: '2026-07-31T00:00:00Z_42',
    })
  })

  it('consumes latest cursor metadata without inferring from item count', async () => {
    server.use(http.get('/api/v1/posts', ({ request }) => {
      const params = new URL(request.url).searchParams
      expect(params.get('sort')).toBe('latest')
      expect(params.get('cursor')).toBe('latest-page-2')
      expect(params.get('offset')).toBeNull()
      return HttpResponse.json({
        data: Array.from({ length: 12 }, (_, id) => ({ id })),
        meta: { nextCursor: '', hasMore: false, requestId: 'latest-final-page' },
      })
    }))

    const page = await fetchFeedPage(baseFilter, 2, 12, 'latest-page-2')

    expect(page.items).toHaveLength(12)
    expect(page.hasMore).toBe(false)
  })

	it('uses ranked cursor metadata for recommended pages', async () => {
		server.use(http.get('/api/v1/posts', ({ request }) => {
			const params = new URL(request.url).searchParams
			expect(params.get('sort')).toBe('recommended')
			expect(params.get('limit')).toBe('12')
			expect(params.get('offset')).toBeNull()
			expect(params.get('cursor')).toBe('recommended-page-2')
			return HttpResponse.json({
				data: Array.from({ length: 12 }, (_, id) => ({ id })),
				meta: { nextCursor: 'recommended-page-3', hasMore: true, requestId: 'recommended-cursor' },
			})
		}))

		const page = await fetchFeedPage({ ...baseFilter, sort: 'recommended' }, 2, 12, 'recommended-page-2')

		expect(page.items).toHaveLength(12)
		expect(page.hasMore).toBe(true)
		expect(page.nextCursor).toBe('recommended-page-3')
	})
})
