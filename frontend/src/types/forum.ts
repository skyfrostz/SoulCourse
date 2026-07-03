export type Track = 'physics' | 'history'
export type Subject = 'chemistry' | 'biology' | 'politics' | 'geography'
export type Category = 'experience' | 'question' | 'data'
export type Role = 'student' | 'parent' | 'teacher' | 'counselor'
export type FeedSort = 'recommended' | 'latest' | 'hot'

export interface Post {
  id: number
  userId?: number
  authorName: string
  authorRole: Role
  title: string
  content: string
  imageUrls: string[]
  tags: string[]
  track: Track
  electives: Subject[]
  category: Category
  grade: string
  province: string
  likesCount: number
  commentsCount: number
  favoritesCount: number
  viewerLiked: boolean
  viewerFavorited: boolean
  viewerFollowing: boolean
  createdAt: string
  updatedAt: string
}

export interface Comment {
  id: number
  postId: number
  userId?: number
  author: string
  role: Role
  content: string
  createdAt: string
}

export interface SubjectInsight {
  id: number
  combination: string
  trend: string
  heat: number
  matchRate: number
  coverageRate?: number
  advice: string
  details: string
  updatedAt: string
}

export interface FeedFilter {
  track: Track
  subjects: Subject[]
  category: Category | 'all'
  keyword: string
  sort: FeedSort
}

export interface User {
  id: number
  email: string
  nickname: string
  role: Role
  province: string
  grade: string
  createdAt: string
}

export interface ChoiceProfile {
  realName: string
  city: string
  schoolType: string
  gradeRank: string
  mbti: string
  targetMajors: string
  targetCities: string
  subjectStability: string
  physicsScore: string
  historyScore: string
  chemistryScore: string
  biologyScore: string
  politicsScore: string
  geographyScore: string
  preferredTrack: Track
  preferredSubjects: Subject[]
  learningStyle: string
  pressureTolerance: string
  recommendationFocus: string
}

export interface AuthSession {
  user: User
  token: string
}

export interface Topic {
  id: number
  slug: string
  title: string
  summary: string
  viewsCount: number
  postsCount: number
  createdAt: string
}

export interface TopicDetail {
  topic: Topic
  posts: Post[]
}

export interface ToggleResult {
  active: boolean
  count: number
}

export interface ChoiceAdvice {
  summary: string
  risks: string[]
  actions: string[]
  querySuggestions: string[]
  source: 'ai' | 'fallback'
}
