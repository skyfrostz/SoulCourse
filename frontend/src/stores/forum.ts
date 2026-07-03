import { defineStore } from 'pinia'
import { authStorageKey } from '../lib/api'
import { notificationSeeds } from '../lib/notifications'
import type {
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

interface LocalFollowState {
  following: Record<string, Record<string, FollowProfile>>
  followers: Record<string, Record<string, FollowProfile>>
}

export const useForumStore = defineStore('forum', {
  state: () => ({
    filter: {
      track: 'physics' as Track,
      subjects: ['chemistry', 'biology'] as Subject[],
      category: 'all' as Category | 'all',
      keyword: '',
      sort: 'recommended' as FeedSort,
    } satisfies FeedFilter,
    selectedPostId: 1,
    page: 1,
    pageSize: 12,
    session: readStoredSession(),
    authOpen: false,
    publishOpen: false,
    publishCategory: 'question' as Category,
    refreshHint: '',
    readNotificationIds: readStoredNotificationReads(),
    messageUnread: 3,
    choiceProfile: readStoredChoiceProfile(),
    localEngagement: readStoredLocalEngagement(),
    detailPanel: { kind: 'none' } as DetailPanel,
  }),
  getters: {
    isAuthed: (state) => Boolean(state.session?.token),
    currentUser: (state) => state.session?.user ?? null,
    unreadNotificationCount: (state) =>
      notificationSeeds.filter((notification) => !state.readNotificationIds[notification.id]).length,
  },
  actions: {
    setTrack(track: Track) {
      this.filter.track = track
      this.page = 1
    },
    toggleSubject(subject: Subject) {
      if (this.filter.subjects.includes(subject)) {
        if (this.filter.subjects.length > 1) {
          this.filter.subjects = this.filter.subjects.filter((item) => item !== subject)
        }
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
        track: 'physics',
        subjects: ['chemistry', 'biology'],
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
    markNotificationsRead(ids?: string[]) {
      const next = { ...this.readNotificationIds }
      const targetIds = ids ?? notificationSeeds.map((notification) => notification.id)
      targetIds.forEach((id) => {
        next[id] = true
      })
      this.readNotificationIds = next
      localStorage.setItem(notificationReadsStorageKey, JSON.stringify(next))
    },
    markMessagesRead(count?: number) {
      const readCount = count ?? this.messageUnread
      this.messageUnread = Math.max(0, this.messageUnread - readCount)
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
    },
    logout() {
      this.session = null
      localStorage.removeItem(authStorageKey)
    },
    requireAuth() {
      if (!this.isAuthed) {
        this.authOpen = true
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
    openTopic(detail: TopicDetail) {
      this.detailPanel = { kind: 'topic', detail }
    },
    openInsight(detail: SubjectInsight) {
      this.detailPanel = { kind: 'insight', detail }
    },
    hydratePost(post: Post): Post {
      const engagement = this.localEngagement.posts[post.id] ?? {}
      const graphFollowing = this.currentUser ? this.isUserFollowing(post.authorName) : undefined
      return { ...post, ...engagement, viewerFollowing: graphFollowing ?? engagement.viewerFollowing ?? post.viewerFollowing }
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
    currentFollowProfile(): FollowProfile | null {
      if (!this.currentUser) return null
      return {
        name: this.currentUser.nickname,
        role: this.currentUser.role,
        province: this.currentUser.province,
        grade: this.currentUser.grade,
        followedAt: new Date().toISOString(),
      }
    },
    isUserFollowing(name: string): boolean {
      const currentName = this.currentUser?.nickname
      if (!currentName) return false
      return Boolean(this.localEngagement.follows.following[currentName]?.[name])
    },
    getFollowing(name: string): FollowProfile[] {
      return Object.values(this.localEngagement.follows.following[name] ?? {}).sort((a, b) => b.followedAt.localeCompare(a.followedAt))
    },
    getFollowers(name: string): FollowProfile[] {
      return Object.values(this.localEngagement.follows.followers[name] ?? {}).sort((a, b) => b.followedAt.localeCompare(a.followedAt))
    },
    setUserFollow(profile: FollowProfile, active: boolean): boolean {
      const current = this.currentFollowProfile()
      if (!current || current.name === profile.name) return false
      const currentFollowing = { ...(this.localEngagement.follows.following[current.name] ?? {}) }
      const targetFollowers = { ...(this.localEngagement.follows.followers[profile.name] ?? {}) }
      if (active) {
        const stampedProfile = { ...profile, followedAt: new Date().toISOString() }
        currentFollowing[profile.name] = stampedProfile
        targetFollowers[current.name] = { ...current, followedAt: stampedProfile.followedAt }
      } else {
        delete currentFollowing[profile.name]
        delete targetFollowers[current.name]
      }
      this.localEngagement.follows.following[current.name] = currentFollowing
      this.localEngagement.follows.followers[profile.name] = targetFollowers
      persistLocalEngagement(this.localEngagement)
      return active
    },
    toggleUserFollow(profile: FollowProfile): boolean {
      if (!this.requireAuth()) return false
      return this.setUserFollow(profile, !this.isUserFollowing(profile.name))
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

export const choiceProfileStorageKey = 'scf_choice_profile'
export const notificationReadsStorageKey = 'scf_notification_reads'

function readStoredSession(): AuthSession | null {
  try {
    const raw = localStorage.getItem(authStorageKey)
    return raw ? (JSON.parse(raw) as AuthSession) : null
  } catch {
    localStorage.removeItem(authStorageKey)
    return null
  }
}

function readStoredChoiceProfile(): ChoiceProfile {
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
    const raw = localStorage.getItem(choiceProfileStorageKey)
    return raw ? { ...defaults, ...(JSON.parse(raw) as ChoiceProfile) } : defaults
  } catch {
    localStorage.removeItem(choiceProfileStorageKey)
    return defaults
  }
}

function readStoredNotificationReads(): Record<string, boolean> {
  try {
    const raw = localStorage.getItem(notificationReadsStorageKey)
    return raw ? (JSON.parse(raw) as Record<string, boolean>) : {}
  } catch {
    localStorage.removeItem(notificationReadsStorageKey)
    return {}
  }
}

export const localEngagementStorageKey = 'scf_local_engagement'

function persistLocalEngagement(state: LocalEngagementState) {
  localStorage.setItem(localEngagementStorageKey, JSON.stringify(state))
}

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
