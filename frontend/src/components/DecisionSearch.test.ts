import { createPinia, setActivePinia } from 'pinia'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import DecisionSearch from './DecisionSearch.vue'

const fetchPublishedRequirements = vi.fn()
const fetchProvinceCoverage = vi.fn()

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return {
    ...actual,
    fetchPublishedRequirements: () => fetchPublishedRequirements(),
    fetchProvinceCoverage: () => fetchProvinceCoverage(),
  }
})

vi.mock('../composables/useGlobalSearch', () => ({
  useGlobalSearch: () => ({ runSearch: vi.fn() }),
}))

const RouterLink = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => { resolve = done })
  return { promise, resolve }
}

function renderSearch() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  return render(DecisionSearch, {
    props: {
      posts: [{
        id: 7,
        title: '临床医学选科经验',
        content: '帖子内容',
        authorName: '广东学长',
        authorRole: 'student',
        imageUrls: [],
        tags: ['临床医学'],
        track: 'physics',
        electives: ['chemistry'],
        category: 'experience',
        grade: '高二',
        province: '广东',
        likesCount: 2,
        commentsCount: 1,
        favoritesCount: 1,
        viewerLiked: false,
        viewerFavorited: false,
        viewerFollowing: false,
        createdAt: '2026-07-31T00:00:00Z',
        updatedAt: '2026-07-31T00:00:00Z',
      }],
      topics: [{
        id: 3,
        slug: 'clinical',
        topicTag: '临床医学',
        title: '临床医学',
        summary: '专业选科讨论',
        viewsCount: 1200,
        postsCount: 5,
        createdAt: '2026-07-31T00:00:00Z',
      }],
    },
    global: {
      plugins: [pinia, [VueQueryPlugin, { queryClient }]],
      stubs: { RouterLink },
    },
  })
}

describe('DecisionSearch', () => {
  afterEach(cleanup)

  beforeEach(() => {
    fetchPublishedRequirements.mockReset()
    fetchProvinceCoverage.mockReset()
    const storage = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
      clear: () => storage.clear(),
    })
  })

  it('keeps posts and topics visible while data sources degrade independently', async () => {
    const provinces = deferred<never[]>()
    fetchPublishedRequirements.mockRejectedValue(new Error('requirements unavailable'))
    fetchProvinceCoverage.mockReturnValue(provinces.promise)

    renderSearch()

    expect(screen.getByRole('link', { name: /临床医学选科经验/ })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /# 临床医学/ })).toBeInTheDocument()
    expect(screen.getByText('正在加载省份政策…')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('专业要求暂时加载失败'))
    expect(screen.queryByText('暂时没有匹配建议')).not.toBeInTheDocument()
  })

  it('retries only the failed requirements source', async () => {
    fetchPublishedRequirements
      .mockRejectedValueOnce(new Error('requirements unavailable'))
      .mockResolvedValueOnce([])
    fetchProvinceCoverage.mockResolvedValue([])

    renderSearch()

    const retryButton = await screen.findByRole('button', { name: '重试专业要求' })
    await fireEvent.click(retryButton)

    await waitFor(() => expect(fetchPublishedRequirements).toHaveBeenCalledTimes(2))
    expect(fetchProvinceCoverage).toHaveBeenCalledTimes(1)
    await waitFor(() => expect(screen.queryByText('专业要求暂时加载失败，其他搜索结果仍可使用。')).not.toBeInTheDocument())
  })
})
