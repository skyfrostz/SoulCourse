import { defineStore } from 'pinia'
import { authStorageKey, fetchMyProfile, fetchNotifications, logout as apiLogout, markAllNotificationsRead, markNotificationRead, registerUnauthorizedHandler } from '../lib/api'
import type {
  AppNotification,
  AuthSession,
  Category,
  ChoiceProfile,
  FeedFilter,
  FeedSort,
  Subject,
  SubjectInsight,
  TopicDetail,
  Track,
  Post,
  Comment,
  Role,
} from '../types/forum'

type DetailPanel =
  | { kind: 'none' }
  | { kind: 'topic'; detail: TopicDetail }
  | { kind: 'insight'; detail: SubjectInsight }

interface LocalPostEngagement {
  viewerLiked?: boolean
  viewerFavorited?: boolean
  viewerFollowing?: boolean
  likesCount?: number
  favoritesCount?: number
  commentsCount?: number
}

interface LocalEngagementState {
  posts: Record<number, LocalPostEngagement>
  follows: LocalFollowState
}

export interface FollowProfile {
  name: string
  role: Role
  province: string
  grade: string
  followedAt: string
}

export const currentProfileAuthRedirect = '@current-profile'

interface LocalFollowState {
  following: Record<string, Record<string, FollowProfile>>
  followers: Record<string, Record<string, FollowProfile>>
}

export const useForumStore = defineStore('forum', {
  state: () => ({
    filter: {
      track: 'all' as Track | 'all',
      subjects: [] as Subject[],
      category: 'all' as Category | 'all',
      keyword: '',
      sort: 'recommended' as FeedSort,
    } satisfies FeedFilter,
    selectedPostId: 1,
    page: 1,
    pageSize: 12,
    session: readStoredSession(),
    authOpen: false,
    authRedirect: '',
    authMessage: '',
    publishOpen: false,
    publishCategory: 'question' as Category,
    refreshHint: '',
    notifications: [] as AppNotification[],
    notificationReadError: '',
    choiceProfile: readStoredChoiceProfile(),
    localEngagement: readStoredLocalEngagement(),
    detailPanel: { kind: 'none' } as DetailPanel,
  }),
  getters: {
    isAuthed: (state) => Boolean(state.session?.user) && (!state.session?.expiresAt || new Date(state.session.expiresAt).getTime() > Date.now()),
    currentUser: (state) => state.session?.user ?? null,
    unreadNotificationCount: (state) => state.notifications.filter((notification) => !notification.readAt).length,
  },
  actions: {
    setTrack(track: Track | 'all') {
      this.filter.track = track
      this.page = 1
    },
    toggleSubject(subject: Subject) {
      if (this.filter.subjects.includes(subject)) {
        this.filter.subjects = this.filter.subjects.filter((item) => item !== subject)
        return
      }
      this.filter.subjects = [...this.filter.subjects.slice(-1), subject]
    },
    setSubjects(subjects: Subject[]) {
      this.filter.subjects = subjects.slice(0, 2)
      this.page = 1
    },
    setCategory(category: Category | 'all') {
      this.filter.category = category
      this.page = 1
    },
    browseCategory(category: Category | 'all') {
      this.filter.category = category
      this.filter.keyword = ''
      this.page = 1
    },
    resetFilters() {
      this.filter = {
        track: 'all',
        subjects: [],
        category: 'all',
        keyword: '',
        sort: 'recommended',
      }
      this.page = 1
    },
    setSort(sort: FeedSort) {
      this.filter.sort = sort
      this.page = 1
    },
    setKeyword(keyword: string) {
      this.filter.keyword = keyword
      this.page = 1
    },
    triggerRefreshHint() {
      this.refreshHint = '正在为你搜寻全网选科秘籍...'
      window.setTimeout(() => {
        this.refreshHint = ''
      }, 1600)
    },
    async markNotificationsRead(ids?: number[]) {
      if (!this.isAuthed) return
      this.notificationReadError = ''
      try {
        if (ids?.length) {
          await Promise.all(ids.map((id) => markNotificationRead(id)))
          this.notifications = this.notifications.map((item) => ids.includes(item.id) ? { ...item, readAt: new Date().toISOString() } : item)
          return
        }
        await markAllNotificationsRead()
        const readAt = new Date().toISOString()
        this.notifications = this.notifications.map((item) => ({ ...item, readAt: item.readAt ?? readAt }))
      } catch {
        this.notificationReadError = '通知状态同步失败，请稍后重试。'
      }
    },
    setPage(page: number) {
      this.page = Math.max(1, page)
    },
    selectPost(postId: number) {
      this.selectedPostId = postId
      this.detailPanel = { kind: 'none' }
    },
    setSession(session: AuthSession) {
      this.session = session
      localStorage.setItem(authStorageKey, JSON.stringify(session))
      this.authOpen = false
      this.authMessage = ''
      void this.hydrateAccount()
    },
    async logout() {
      this.session = null
      this.notifications = []
      localStorage.removeItem(choiceProfileStorageKey)
      this.choiceProfile = readStoredChoiceProfile(true)
      localStorage.removeItem(authStorageKey)
      this.authRedirect = ''
      try {
        await apiLogout()
      } catch {
        // Local sign-out must still complete if the current cookie session is already invalid.
      }
    },
    handleUnauthorized() {
      if (!this.session) return
      void this.logout()
      this.authMessage = '登录状态已失效，请重新登录后继续。'
      this.openAuth()
    },
    openAuth(redirect = '') {
      this.authRedirect = redirect
      this.authOpen = true
    },
    requireAuth(redirect = '') {
      if (!this.isAuthed) {
        if (this.session) this.authMessage = '登录状态已过期，请重新登录后继续。'
        this.openAuth(redirect)
        return false
      }
      return true
    },
    openPublish(category: Category = 'question') {
      if (this.requireAuth()) {
        this.publishCategory = category
        this.publishOpen = true
      }
    },
    saveChoiceProfile(profile: ChoiceProfile) {
      this.choiceProfile = profile
      localStorage.setItem(choiceProfileStorageKey, JSON.stringify(profile))
    },
    async hydrateAccount() {
      if (!this.isAuthed) return
      try {
        const [profile, notifications] = await Promise.all([fetchMyProfile(), fetchNotifications()])
        this.choiceProfile = profile.choiceProfile
        this.notifications = notifications.items
        localStorage.setItem(choiceProfileStorageKey, JSON.stringify(profile.choiceProfile))
      } catch {
        this.notifications = []
      }
    },
    openTopic(detail: TopicDetail) {
      this.detailPanel = { kind: 'topic', detail }
    },
    openInsight(detail: SubjectInsight) {
      this.detailPanel = { kind: 'insight', detail }
    },
    hydratePost(post: Post): Post {
      const engagement = this.localEngagement.posts[post.id] ?? {}
      return { ...post, ...engagement, viewerFollowing: post.viewerFollowing }
    },
    hydratePosts(posts: Post[]): Post[] {
      return posts.map((post) => this.hydratePost(post))
    },
    getPostComments(postId: number, fallback: Comment[] = []): Comment[] {
      void postId
      return fallback
    },
    getActualCommentCount(postId: number, fallback: Comment[] = []): number {
      return this.getPostComments(postId, fallback).length
    },
    getFavoritePosts(fallbackPosts: Post[] = []): Post[] {
      const merged = new Map<number, Post>()
      fallbackPosts.forEach((post) => {
        const hydrated = this.hydratePost(post)
        if (hydrated.viewerFavorited) merged.set(post.id, hydrated)
      })
      return Array.from(merged.values()).sort((a, b) => b.id - a.id)
    },
    closeDetail() {
      this.detailPanel = { kind: 'none' }
    },
  },
})

registerUnauthorizedHandler(() => {
  const forumStore = useForumStore()
  forumStore.handleUnauthorized()
})

export const choiceProfileStorageKey = 'scf_choice_profile'

function readStoredSession(): AuthSession | null {
  try {
    const raw = localStorage.getItem(authStorageKey)
    if (!raw) return null
    const session = JSON.parse(raw) as AuthSession
    if (session.expiresAt && new Date(session.expiresAt).getTime() <= Date.now()) {
      localStorage.removeItem(authStorageKey)
      return null
    }
    return session
  } catch {
    localStorage.removeItem(authStorageKey)
    return null
  }
}

function readStoredChoiceProfile(reset = false): ChoiceProfile {
  const defaults: ChoiceProfile = {
    realName: '',
    city: '',
    schoolType: '普通高中',
    gradeRank: '',
    mbti: '',
    targetMajors: '',
    targetCities: '',
    subjectStability: '中等',
    physicsScore: '',
    historyScore: '',
    chemistryScore: '',
    biologyScore: '',
    politicsScore: '',
    geographyScore: '',
    preferredTrack: 'physics',
    preferredSubjects: ['chemistry', 'biology'],
    learningStyle: '理解推导型',
    pressureTolerance: '中等',
    recommendationFocus: '专业覆盖率优先',
  }

  try {
    const raw = reset ? null : localStorage.getItem(choiceProfileStorageKey)
    return raw ? { ...defaults, ...(JSON.parse(raw) as ChoiceProfile) } : defaults
  } catch {
    localStorage.removeItem(choiceProfileStorageKey)
    return defaults
  }
}

export const localEngagementStorageKey = 'scf_local_engagement'

function readStoredLocalEngagement(): LocalEngagementState {
  try {
    const raw = localStorage.getItem(localEngagementStorageKey)
    if (!raw) return createEmptyLocalEngagement()
    const parsed = JSON.parse(raw) as LocalEngagementState
    return {
      posts: parsed.posts ?? {},
      follows: {
        following: parsed.follows?.following ?? {},
        followers: parsed.follows?.followers ?? {},
      },
    }
  } catch {
    localStorage.removeItem(localEngagementStorageKey)
    return createEmptyLocalEngagement()
  }
}

function createEmptyLocalEngagement(): LocalEngagementState {
  return { posts: {}, follows: { following: {}, followers: {} } }
}
