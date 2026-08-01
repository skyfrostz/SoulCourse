import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia, setActivePinia } from 'pinia'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { defineComponent } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useForumStore } from '../stores/forum'
import { useForumData } from './useForumData'

const { fetchFeedPage, fetchInsights, fetchTopics } = vi.hoisted(() => ({
  fetchFeedPage: vi.fn(),
  fetchInsights: vi.fn(() => Promise.resolve([])),
  fetchTopics: vi.fn(() => Promise.resolve([])),
}))

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return {
    ...actual,
    fetchFeedPage,
    fetchInsights,
    fetchTopics,
  }
})

const Harness = defineComponent({
  setup() {
    const forumStore = useForumStore()
    const { posts, hasMore } = useForumData()
    return { forumStore, posts, hasMore }
  },
  template: `
    <span data-testid="post-count">{{ posts.length }}</span>
    <button :disabled="!hasMore" @click="forumStore.setPage(forumStore.page + 1)">下一页</button>
  `,
})

describe('useForumData pagination', () => {
  beforeEach(() => {
    const storage = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
      clear: () => storage.clear(),
    })
    fetchFeedPage.mockReset()
    fetchInsights.mockClear()
    fetchTopics.mockClear()
    setActivePinia(createPinia())
  })

  afterEach(() => cleanup())

  it('passes the API nextCursor into the next latest-page request', async () => {
    fetchFeedPage
      .mockResolvedValueOnce({ items: [{ id: 1 }], nextCursor: 'cursor-page-2', hasMore: true })
      .mockResolvedValueOnce({ items: [{ id: 2 }], hasMore: false })
    const pinia = createPinia()
    setActivePinia(pinia)
    const forumStore = useForumStore()
    forumStore.setSort('latest')
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })

    render(Harness, {
      global: { plugins: [pinia, [VueQueryPlugin, { queryClient }]] },
    })

    await waitFor(() => expect(screen.getByRole('button', { name: '下一页' })).toBeEnabled())
    await fireEvent.click(screen.getByRole('button', { name: '下一页' }))
    await waitFor(() => expect(fetchFeedPage).toHaveBeenCalledTimes(2))

    expect(fetchFeedPage.mock.calls[1][1]).toBe(2)
    expect(fetchFeedPage.mock.calls[1][3]).toBe('cursor-page-2')
    await waitFor(() => expect(screen.getByRole('button', { name: '下一页' })).toBeDisabled())
  })

  it('disables next page from hasMore even when the current page is full', async () => {
    fetchFeedPage.mockResolvedValue({
      items: Array.from({ length: 12 }, (_, index) => ({ id: index + 1 })),
      hasMore: false,
    })
    const pinia = createPinia()
    setActivePinia(pinia)
    const forumStore = useForumStore()
    forumStore.setSort('latest')
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })

    render(Harness, {
      global: { plugins: [pinia, [VueQueryPlugin, { queryClient }]] },
    })

    await waitFor(() => expect(screen.getByTestId('post-count')).toHaveTextContent('12'))
    expect(screen.getByRole('button', { name: '下一页' })).toBeDisabled()
  })
})
