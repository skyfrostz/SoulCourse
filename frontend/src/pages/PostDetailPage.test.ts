import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { render, waitFor } from '@testing-library/vue'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PostDetailPage from './PostDetailPage.vue'

const fetchPostDetail = vi.fn()

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return {
    ...actual,
    fetchPostDetail: (postId: number) => fetchPostDetail(postId),
  }
})

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: {} }),
  useRouter: () => ({ push: vi.fn() }),
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
}))

describe('PostDetailPage modal layout', () => {
  beforeEach(() => {
    const storage = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
      clear: () => storage.clear(),
    })
    fetchPostDetail.mockReset()
    fetchPostDetail.mockResolvedValue({
      post: {
        id: 97,
        authorName: '高一新用户',
        authorRole: 'student',
        title: '物化生在高考中会比物化地吃香吗？',
        content: '比较生物和地理的赋分与专业选择。',
        imageUrls: [],
        tags: ['物理方向', '物化生'],
        track: 'physics',
        electives: ['chemistry', 'biology'],
        category: 'question',
        grade: '高一',
        province: '广东',
        likesCount: 0,
        commentsCount: 0,
        favoritesCount: 0,
        viewerLiked: false,
        viewerFavorited: false,
        viewerFollowing: false,
        createdAt: '2026-08-03T00:00:00Z',
        updatedAt: '2026-08-03T00:00:00Z',
      },
      comments: [],
    })
  })

  it('uses the shared two-column modal structure for posts without images', async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const pinia = createPinia()
    setActivePinia(pinia)
    const { container } = render(PostDetailPage, {
      props: { postId: 97, mode: 'modal' },
      global: {
        plugins: [pinia, [VueQueryPlugin, { queryClient }]],
        stubs: {
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
        },
      },
    })

    await waitFor(() => expect(fetchPostDetail).toHaveBeenCalledWith(97))
    await waitFor(() => expect(container.querySelector('.article-layout')).toHaveClass('has-images', 'has-generated-cover'))

    const mediaColumn = container.querySelector('.article-media-column')
    const contentColumn = container.querySelector('.article-content-column')
    expect(mediaColumn?.querySelector('.article-generated-media-cover')).toBeInTheDocument()
    expect(container.querySelectorAll('.article-title-cover')).toHaveLength(1)
    expect(contentColumn?.querySelector('#post-comments')).toBeInTheDocument()
  })
})
