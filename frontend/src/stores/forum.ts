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
} from '../types/forum'

type DetailPanel =
  | { kind: 'none' }
  | { kind: 'topic'; detail: TopicDetail }
  | { kind: 'insight'; detail: SubjectInsight }

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
    pageSize: 4,
    session: readStoredSession(),
    authOpen: false,
    publishOpen: false,
    publishCategory: 'question' as Category,
    refreshHint: '',
    notificationUnread: 12,
    messageUnread: 3,
    choiceProfile: readStoredChoiceProfile(),
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
    setCategory(category: Category | 'all') {
      this.filter.category = category
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
