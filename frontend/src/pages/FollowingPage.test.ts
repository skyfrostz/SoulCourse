import { createPinia, setActivePinia } from 'pinia'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FollowingPage from './FollowingPage.vue'
import { useForumStore } from '../stores/forum'

const fetchMyProfile = vi.fn()
const routerBack = vi.fn()
let pinia: ReturnType<typeof createPinia>

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return {
    ...actual,
    fetchMyProfile: () => fetchMyProfile(),
  }
})

vi.mock('vue-router', () => ({
  useRouter: () => ({ back: routerBack }),
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
}))

describe('FollowingPage', () => {
  beforeEach(() => {
    fetchMyProfile.mockResolvedValue({
      user: {
        id: 1,
        publicId: 'u_1',
        email: 'student@example.com',
        nickname: '广东学生',
        role: 'student',
        province: '广东',
        grade: '高一',
        createdAt: '2026-07-31T00:00:00Z',
      },
      bio: '',
      choiceProfile: {
        realName: '',
        city: '',
        schoolType: '',
        gradeRank: '',
        mbti: '',
        targetMajors: '',
        targetCities: '',
        subjectStability: '',
        physicsScore: '',
        historyScore: '',
        chemistryScore: '',
        biologyScore: '',
        politicsScore: '',
        geographyScore: '',
        preferredTrack: 'physics',
        preferredSubjects: [],
        learningStyle: '',
        pressureTolerance: '',
        recommendationFocus: '',
      },
      stats: { posts: 0, comments: 0, following: 1, followers: 1, favorites: 0, engagement: 0 },
      posts: [],
      comments: [],
      favorites: [],
      viewerFollowing: false,
      following: [{
        name: '规划老师',
        role: 'teacher',
        province: '广东',
        grade: '高中',
        followedAt: '2026-07-31T00:00:00Z',
      }],
      followers: [{
        name: '学长',
        role: 'student',
        province: '广东',
        grade: '高二',
        followedAt: '2026-07-31T00:00:00Z',
      }],
    })
    const storage = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
      clear: () => storage.clear(),
    })
    pinia = createPinia()
    setActivePinia(pinia)
    const store = useForumStore()
    store.session = {
      user: {
        id: 1,
        publicId: 'u_1',
        email: 'student@example.com',
        nickname: '广东学生',
        role: 'student',
        province: '广东',
        grade: '高一',
        createdAt: '2026-07-31T00:00:00Z',
      },
      expiresAt: '2099-01-01T00:00:00Z',
    }
  })

  it('shows a filtered empty state when the search has no match', async () => {
    render(FollowingPage, {
      global: {
        plugins: [pinia, VueQueryPlugin],
        stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } },
      },
    })

    await waitFor(() => expect(screen.getByRole('link', { name: /规划老师/ })).toBeInTheDocument())
    await fireEvent.update(screen.getByPlaceholderText('搜索姓名、身份、省份...'), '不存在')

    expect(screen.getByRole('heading', { name: '没有匹配的关注用户' })).toBeInTheDocument()
    expect(screen.getByText('试试清空搜索词，或者换一个姓名、省份、身份关键词。')).toBeInTheDocument()
  })
})
