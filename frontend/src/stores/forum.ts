import { defineStore } from 'pinia'
import { authStorageKey } from '../lib/api'
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
  createdPosts: Record<number, Post>
  comments: Record<number, Comment[]>
  favoritePosts: Record<number, Post>
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
    notificationUnread: 12,
    messageUnread: 3,
    choiceProfile: readStoredChoiceProfile(),
    localEngagement: readStoredLocalEngagement(),
    detailPanel: { kind: 'none' } as DetailPanel,
  }),
  getters: {
    isAuthed: (state) => Boolean(state.session?.token),
    currentUser: (state) => state.session?.user ?? null,
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
    markNotificationsRead() {
      this.notificationUnread = 0
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
      return [...fallback, ...(this.localEngagement.comments[postId] ?? [])]
    },
    getLocalComments(): Comment[] {
      return Object.values(this.localEngagement.comments).flat()
    },
    getCreatedPosts(): Post[] {
      return Object.values(this.localEngagement.createdPosts).sort(
        (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
      )
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
      Object.values(this.localEngagement.favoritePosts).forEach((post) => {
        const hydrated = this.hydratePost(post)
        if (hydrated.viewerFavorited) merged.set(post.id, hydrated)
      })
      return Array.from(merged.values()).sort((a, b) => b.id - a.id)
    },
    toggleLocalLike(post: Post): Post {
      const current = this.hydratePost(post)
      const nextLiked = !current.viewerLiked
      const nextPost = {
        ...current,
        viewerLiked: nextLiked,
        likesCount: Math.max(0, current.likesCount + (nextLiked ? 1 : -1)),
      }
      this.localEngagement.posts[post.id] = pickLocalPostEngagement(nextPost)
      persistLocalEngagement(this.localEngagement)
      return nextPost
    },
    toggleLocalFavorite(post: Post): Post {
      const current = this.hydratePost(post)
      const nextFavorited = !current.viewerFavorited
      const nextPost = {
        ...current,
        viewerFavorited: nextFavorited,
        favoritesCount: Math.max(0, current.favoritesCount + (nextFavorited ? 1 : -1)),
      }
      this.localEngagement.posts[post.id] = pickLocalPostEngagement(nextPost)
      if (nextFavorited) {
        this.localEngagement.favoritePosts[post.id] = nextPost
      } else {
        delete this.localEngagement.favoritePosts[post.id]
      }
      persistLocalEngagement(this.localEngagement)
      return nextPost
    },
    toggleLocalFollow(post: Post): Post {
      const current = this.hydratePost(post)
      const active = this.toggleUserFollow({
        name: post.authorName,
        role: post.authorRole,
        province: post.province,
        grade: post.grade,
        followedAt: new Date().toISOString(),
      })
      const nextPost = { ...current, viewerFollowing: active }
      this.localEngagement.posts[post.id] = pickLocalPostEngagement(nextPost)
      persistLocalEngagement(this.localEngagement)
      return nextPost
    },
    addLocalComment(postId: number, content: string, basePost?: Post): Comment {
      const author = this.currentUser?.nickname ?? '演示同学'
      const role = this.currentUser?.role ?? 'student'
      const comment: Comment = {
        id: Date.now(),
        postId,
        userId: this.currentUser?.id,
        author,
        role,
        content,
        createdAt: new Date().toISOString(),
      }
      this.localEngagement.comments[postId] = [...(this.localEngagement.comments[postId] ?? []), comment]
      const current = this.localEngagement.posts[postId] ?? {}
      const currentCount = current.commentsCount ?? basePost?.commentsCount ?? 0
      this.localEngagement.posts[postId] = {
        ...current,
        commentsCount: currentCount + 1,
      }
      persistLocalEngagement(this.localEngagement)
      return comment
    },
    createLocalPost(payload: {
      title: string
      content: string
      imageUrls: string[]
      tags: string[]
      track: Track
      electives: Subject[]
      category: Category
      grade: string
      province: string
    }): Post {
      const now = new Date().toISOString()
      const id = Date.now() + Math.floor(Math.random() * 1000)
      const current = this.currentUser
      const post: Post = {
        id,
        userId: current?.id,
        authorName: current?.nickname ?? '演示同学',
        authorRole: current?.role ?? 'student',
        title: payload.title.trim(),
        content: payload.content.trim(),
        imageUrls: payload.imageUrls.slice(0, 9),
        tags: payload.tags.slice(0, 8),
        track: payload.track,
        electives: payload.electives.slice(0, 2),
        category: payload.category,
        grade: payload.grade,
        province: payload.province,
        likesCount: 0,
        commentsCount: 0,
        favoritesCount: 0,
        viewerLiked: false,
        viewerFavorited: false,
        viewerFollowing: false,
        createdAt: now,
        updatedAt: now,
      }
      this.localEngagement.createdPosts[id] = post
      this.localEngagement.posts[id] = pickLocalPostEngagement(post)
      persistLocalEngagement(this.localEngagement)
      return post
    },
    closeDetail() {
      this.detailPanel = { kind: 'none' }
    },
  },
})

export const choiceProfileStorageKey = 'scf_choice_profile'

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

export const localEngagementStorageKey = 'scf_local_engagement'

function pickLocalPostEngagement(post: Post): LocalPostEngagement {
  return {
    viewerLiked: post.viewerLiked,
    viewerFavorited: post.viewerFavorited,
    viewerFollowing: post.viewerFollowing,
    likesCount: post.likesCount,
    favoritesCount: post.favoritesCount,
    commentsCount: post.commentsCount,
  }
}

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
      createdPosts: parsed.createdPosts ?? {},
      comments: parsed.comments ?? {},
      favoritePosts: parsed.favoritePosts ?? {},
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
  return { posts: {}, createdPosts: {}, comments: {}, favoritePosts: {}, follows: { following: {}, followers: {} } }
}
