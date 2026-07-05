import axios from 'axios'
import { defaultApiBasePath } from './runtime'
import type {
  AuthSession,
  ChoiceAdvice,
  ChoiceProfile,
  Comment,
  FeedFilter,
  Post,
  SubjectInsight,
  ToggleResult,
  Topic,
  TopicDetail,
  User,
} from '../types/forum'

export const authStorageKey = 'scf_auth_session'
export const apiDataEnabled = true

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? defaultApiBasePath(),
  timeout: 8000,
})

api.interceptors.request.use((config) => {
  const raw = localStorage.getItem(authStorageKey)
  if (raw) {
    const session = JSON.parse(raw) as AuthSession
    config.headers.Authorization = `Bearer ${session.token}`
  }
  return config
})

interface ApiEnvelope<T> {
  data: T
}

export interface PublishedContentRecord {
  id: string
  module: string
  title: string
  type: string
  status: string
  scope: string
  owner: string
  tags: string[]
  summary: string
  url: string
  priority: string
  sortOrder: number
  payload: {
    imageUrls?: string[]
    [key: string]: unknown
  }
}

export async function register(payload: {
  email: string
  password: string
  verificationCode: string
  nickname: string
  role: string
  province: string
  grade: string
}): Promise<AuthSession> {
  const response = await api.post<ApiEnvelope<AuthSession>>('/auth/register', payload)
  return response.data.data
}

export async function sendEmailVerificationCode(email: string): Promise<{
  email: string
  expiresInSeconds: number
  debugCode?: string
}> {
  const response = await api.post<ApiEnvelope<{ email: string; expiresInSeconds: number; debugCode?: string }>>(
    '/auth/email-verification-code',
    { email },
  )
  return response.data.data
}

export async function login(email: string, password: string): Promise<AuthSession> {
  const response = await api.post<ApiEnvelope<AuthSession>>('/auth/login', { email, password })
  return response.data.data
}

export async function fetchMe(): Promise<User> {
  const response = await api.get<ApiEnvelope<User>>('/me')
  return response.data.data
}

export async function fetchPosts(filter: FeedFilter, page = 1, pageSize = 4): Promise<Post[]> {
  const response = await api.get<ApiEnvelope<Post[]>>('/posts', {
    params: {
      track: filter.track,
      subjects: filter.subjects.join(','),
      category: filter.category === 'all' ? undefined : filter.category,
      q: filter.keyword || undefined,
      sort: filter.sort,
      limit: pageSize,
      offset: (page - 1) * pageSize,
    },
  })
  return response.data.data
}

export async function createPost(payload: {
  title: string
  content: string
  imageUrls: string[]
  tags: string[]
  track: string
  electives: string[]
  category: string
  grade: string
  province: string
}): Promise<Post> {
  const response = await api.post<ApiEnvelope<Post>>('/posts', payload)
  return response.data.data
}

export async function fetchPublishedContent(module?: string): Promise<PublishedContentRecord[]> {
  const response = await api.get<ApiEnvelope<{ records: PublishedContentRecord[] }>>('/content', {
    params: { module },
  })
  return response.data.data.records
}

export async function fetchInsights(): Promise<SubjectInsight[]> {
  const response = await api.get<ApiEnvelope<SubjectInsight[]>>('/insights')
  return response.data.data
}

export async function fetchInsight(id: number): Promise<SubjectInsight> {
  const response = await api.get<ApiEnvelope<SubjectInsight>>(`/insights/${id}`)
  return response.data.data
}

export async function fetchTopics(): Promise<Topic[]> {
  const response = await api.get<ApiEnvelope<Topic[]>>('/topics')
  return response.data.data
}

export async function fetchTopic(slug: string): Promise<TopicDetail> {
  const response = await api.get<ApiEnvelope<TopicDetail>>(`/topics/${slug}`)
  return response.data.data
}

export async function fetchPostDetail(postId: number): Promise<{ post: Post; comments: Comment[] }> {
  const response = await api.get<ApiEnvelope<{ post: Post; comments: Comment[] }>>(`/posts/${postId}`)
  return response.data.data
}

export async function createComment(postId: number, content: string): Promise<Comment> {
  const response = await api.post<ApiEnvelope<Comment>>(`/posts/${postId}/comments`, { content })
  return response.data.data
}

export async function togglePostLike(postId: number): Promise<ToggleResult> {
  const response = await api.post<ApiEnvelope<ToggleResult>>(`/posts/${postId}/like`)
  return response.data.data
}

export async function togglePostFavorite(postId: number): Promise<ToggleResult> {
  const response = await api.post<ApiEnvelope<ToggleResult>>(`/posts/${postId}/favorite`)
  return response.data.data
}

export async function toggleFollowAuthor(authorName: string): Promise<{ active: boolean }> {
  const response = await api.post<ApiEnvelope<{ active: boolean }>>(`/authors/${encodeURIComponent(authorName)}/follow`)
  return response.data.data
}

export async function requestChoiceAdvice(profile: ChoiceProfile, question = ''): Promise<ChoiceAdvice> {
  const response = await api.post<ApiEnvelope<ChoiceAdvice>>('/ai/choice-advice', { profile, question })
  return response.data.data
}
