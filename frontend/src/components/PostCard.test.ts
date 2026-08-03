import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { fireEvent, render } from '@testing-library/vue'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PostCard from './PostCard.vue'
import type { Post } from '../types/forum'

const { routerPush } = vi.hoisted(() => ({ routerPush: vi.fn() }))

vi.mock('vue-router', () => ({
  useRoute: () => ({ name: 'community', query: { source: 'welcome' } }),
  useRouter: () => ({ push: routerPush }),
}))

const post: Post = {
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
}

describe('PostCard community entry', () => {
  beforeEach(() => {
    routerPush.mockReset()
    const storage = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
      clear: () => storage.clear(),
    })
  })

  it('opens the first post click in the feed modal after entering from welcome', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const { container } = render(PostCard, {
      props: { post },
      global: {
        plugins: [pinia, [VueQueryPlugin, { queryClient }]],
        stubs: {
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
        },
      },
    })

    await fireEvent.click(container.querySelector('.post-title-cover') as HTMLElement)

    expect(routerPush).toHaveBeenCalledOnce()
    expect(routerPush).toHaveBeenCalledWith({
      name: 'home',
      query: { source: 'welcome', post: '97' },
    })
  })
})
