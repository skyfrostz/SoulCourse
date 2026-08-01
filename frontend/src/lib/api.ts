import axios from 'axios'
import { defaultApiBasePath } from './runtime'
import { type ApiEnvelope, normalizeApiError } from './api-contract'
import type {
  AccountProfile,
  AccountSession,
  AppNotification,
  AuthSession,
  ChoiceAdvice,
  ChoiceProfile,
  Comment,
  ContentReport,
  Conversation,
  DirectMessage,
  FeedFilter,
  Post,
  SubjectInsight,
  Taxonomy,
  ToggleResult,
  Topic,
  TopicDetail,
  User,
} from '../types/forum'

export const authStorageKey = 'scf_auth_session'
export const apiDataEnabled = true

type UnauthorizedHandler = () => void

const configuredApiBaseUrl = import.meta.env.VITE_API_BASE_URL?.trim()

const api = axios.create({
  baseURL: configuredApiBaseUrl || defaultApiBasePath(),
  timeout: 8000,
  withCredentials: true,
})

let unauthorizedHandler: UnauthorizedHandler | null = null

export function registerUnauthorizedHandler(handler: UnauthorizedHandler) {
  unauthorizedHandler = handler
}

api.interceptors.request.use((config) => {
  const csrfToken = readCookie('scf_csrf')
  const method = config.method?.toUpperCase() ?? 'GET'
  if (csrfToken && !['GET', 'HEAD', 'OPTIONS'].includes(method)) {
    config.headers['X-CSRF-Token'] = csrfToken
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error: unknown) => {
    const normalized = normalizeApiError(error)
    if (normalized.status === 401 && normalized.code === 'unauthorized') {
      unauthorizedHandler?.()
    }
    return Promise.reject(normalized)
  },
)

async function requestData<T>(request: Promise<{ data: ApiEnvelope<T> }>): Promise<T> {
  const response = await request
  return response.data.data
}

function readCookie(name: string): string {
  const prefix = `${encodeURIComponent(name)}=`
  return document.cookie
    .split(';')
    .map((item) => item.trim())
    .find((item) => item.startsWith(prefix))
    ?.slice(prefix.length) ?? ''
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

export interface RealDataRecord {
  id: string
  title: string
  type: string
  scope: string
  coverageStatus: 'verified' | 'unverified'
  dataYear: number
  capturedAt: string
  source: { name: string; url: string }
  fileHash: string
  methodology: string
  summary: string
  tags: string[]
  url: string
  requiredSubjects?: string[]
}

export interface ProvinceCoverage {
  province: string
  coverageStatus: 'verified' | 'unverified'
  recordsCount: number
  dataYear: number
  capturedAt: string
  methodology: string
}

export async function fetchProvinceCoverage(): Promise<ProvinceCoverage[]> {
  const result = await requestData(api.get<ApiEnvelope<{ provinces: ProvinceCoverage[] }>>('/provinces'))
  return result.provinces
}

export async function fetchPublishedPolicies(): Promise<RealDataRecord[]> {
  const result = await requestData(api.get<ApiEnvelope<{ policies: RealDataRecord[] }>>('/policies'))
  return result.policies
}

export async function fetchPublishedRequirements(): Promise<RealDataRecord[]> {
  const result = await requestData(api.get<ApiEnvelope<{ requirements: RealDataRecord[] }>>('/requirements'))
  return result.requirements
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
  return requestData(api.post<ApiEnvelope<AuthSession>>('/auth/register', payload))
}

export async function sendEmailVerificationCode(email: string): Promise<EmailVerificationCodeResult> {
  return requestData(api.post<ApiEnvelope<EmailVerificationCodeResult>>('/auth/email-verification-code', { email }))
}

export async function forgotPassword(email: string): Promise<EmailVerificationCodeResult> {
  return requestData(api.post<ApiEnvelope<EmailVerificationCodeResult>>('/auth/forgot-password', { email }))
}

export async function login(email: string, password: string): Promise<AuthSession> {
  return requestData(api.post<ApiEnvelope<AuthSession>>('/auth/login', { email, password }))
}

export async function resetPassword(payload: { email: string; verificationCode: string; password: string }): Promise<void> {
  await requestData(api.post<ApiEnvelope<{ reset: boolean }>>('/auth/reset-password', payload))
}

export async function logout(token?: string): Promise<void> {
  await requestData(api.post<ApiEnvelope<{ signedOut: boolean }>>('/auth/logout', undefined, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  }))
}

export async function fetchMe(): Promise<User> {
  return requestData(api.get<ApiEnvelope<User>>('/me'))
}

export async function deleteMyAccount(password: string): Promise<void> {
  await requestData(api.delete<ApiEnvelope<{ deleted: boolean }>>('/me', { data: { password } }))
}

export interface EmailVerificationCodeResult {
  email: string
  expiresInSeconds: number
  retryAfterSeconds: number
  hourlyLimit: number
  hourlyRemaining: number
  debugCode?: string
}

export async function fetchMyProfile(): Promise<AccountProfile> {
  return requestData(api.get<ApiEnvelope<AccountProfile>>('/me/profile'))
}

export async function fetchMySessions(): Promise<AccountSession[]> {
  return requestData(api.get<ApiEnvelope<AccountSession[]>>('/me/sessions'))
}

export async function revokeMySession(id: number): Promise<void> {
  await requestData(api.delete<ApiEnvelope<{ revoked: boolean }>>(`/me/sessions/${id}`))
}

export async function fetchProfile(name: string): Promise<AccountProfile> {
  return requestData(api.get<ApiEnvelope<AccountProfile>>(`/profiles/${encodeURIComponent(name)}`))
}

export async function updateMyProfile(payload: { bio: string; choiceProfile: ChoiceProfile }): Promise<AccountProfile> {
  return requestData(api.put<ApiEnvelope<AccountProfile>>('/me/profile', payload))
}

export interface NotificationPage {
  items: AppNotification[]
  nextCursor?: string
  hasMore: boolean
}

export async function fetchNotifications(query: { limit?: number; cursor?: string } = {}): Promise<NotificationPage> {
  const response = await api.get<ApiEnvelope<AppNotification[]>>('/notifications', { params: query })
  return {
    items: response.data.data,
    nextCursor: response.data.meta?.nextCursor,
    hasMore: response.data.meta?.hasMore ?? false,
  }
}

export async function markNotificationRead(id: number): Promise<void> {
  await requestData(api.post<ApiEnvelope<{ read: boolean }>>(`/notifications/${id}/read`))
}

export async function markAllNotificationsRead(): Promise<void> {
  await requestData(api.post<ApiEnvelope<{ read: boolean }>>('/notifications/read-all'))
}

export async function fetchPosts(filter: FeedFilter, _page = 1, pageSize = 4): Promise<Post[]> {
  return fetchPostCollection({
    track: filter.track === 'all' ? undefined : filter.track,
    subjects: filter.subjects,
    category: filter.category === 'all' ? undefined : filter.category,
    province: filter.province,
    tag: filter.tag,
    q: filter.keyword || undefined,
    sort: filter.sort,
    limit: pageSize,
  })
}

export interface FeedPage {
  items: Post[]
  nextCursor?: string
  hasMore: boolean
}

export async function fetchFeedPage(filter: FeedFilter, _page = 1, pageSize = 4, cursor?: string): Promise<FeedPage> {
  const result = await fetchPostCollectionPage({
    track: filter.track === 'all' ? undefined : filter.track,
    subjects: filter.subjects,
    category: filter.category === 'all' ? undefined : filter.category,
    province: filter.province,
    tag: filter.tag,
    q: filter.keyword || undefined,
    sort: filter.sort,
    limit: pageSize,
    cursor,
  })
  return {
    items: result.items,
    nextCursor: result.nextCursor,
    hasMore: result.hasMore && Boolean(result.nextCursor),
  }
}

export interface PostCollectionQuery {
  track?: Exclude<FeedFilter['track'], 'all'>
  subjects?: FeedFilter['subjects']
  category?: Exclude<FeedFilter['category'], 'all'>
  province?: string
  tag?: string
  q?: string
  sort?: FeedFilter['sort']
  limit?: number
  cursor?: string
}

export async function fetchPostCollection(query: PostCollectionQuery = {}): Promise<Post[]> {
  return requestData(api.get<ApiEnvelope<Post[]>>('/posts', {
    params: postCollectionParams(query),
  }))
}

export async function fetchPostCollectionPage(query: PostCollectionQuery = {}): Promise<{ items: Post[]; nextCursor?: string; hasMore: boolean }> {
  const response = await api.get<ApiEnvelope<Post[]>>('/posts', {
    params: postCollectionParams(query),
  })
  return {
    items: response.data.data,
    nextCursor: response.data.meta?.nextCursor,
    hasMore: response.data.meta?.hasMore ?? false,
  }
}

export function postCollectionParams(query: PostCollectionQuery = {}) {
  return {
    ...query,
    subjects: query.subjects?.join(','),
  }
}

export async function fetchTaxonomy(): Promise<Taxonomy> {
  return requestData(api.get<ApiEnvelope<Taxonomy>>('/taxonomy'))
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
  return requestData(api.post<ApiEnvelope<Post>>('/posts', payload))
}

export async function updatePost(postId: number, payload: {
  title: string
  content: string
  tags: string[]
  track: string
  electives: string[]
  category: string
}): Promise<Post> {
  return requestData(api.put<ApiEnvelope<Post>>(`/posts/${postId}`, payload))
}

export async function deletePost(postId: number): Promise<void> {
  await requestData(api.delete<ApiEnvelope<{ deleted: boolean }>>(`/posts/${postId}`))
}

export interface PresignedImageUpload {
  id: string
  assetKey: string
  uploadUrl: string
  method: 'PUT'
  contentType: string
  maxBytes: number
  expiresAt: string
}

export interface CompleteImageUploadResult {
  id: string
  assetKey: string
  url: string
  contentType: string
  sizeBytes: number
  width: number
  height: number
}

export async function uploadImage(file: File, dimensions: { width: number; height: number }): Promise<CompleteImageUploadResult> {
  const upload = await requestData(api.post<ApiEnvelope<PresignedImageUpload>>('/uploads/images/presign', {
    fileName: file.name,
    contentType: file.type,
    sizeBytes: file.size,
    width: dimensions.width,
    height: dimensions.height,
  }))
  await api.put(upload.uploadUrl, file, {
    headers: { 'Content-Type': upload.contentType },
    maxBodyLength: upload.maxBytes,
    // Mobile uploads commonly need longer than the API's normal request timeout.
    timeout: 45_000,
  })
  return requestData(api.post<ApiEnvelope<CompleteImageUploadResult>>(`/uploads/images/${encodeURIComponent(upload.id)}/complete`))
}

export async function fetchPublishedContent(module?: string): Promise<PublishedContentRecord[]> {
  const data = await requestData(api.get<ApiEnvelope<{ records: PublishedContentRecord[] }>>('/content', {
    params: { module },
  }))
  return data.records
}

export async function fetchInsights(): Promise<SubjectInsight[]> {
  return requestData(api.get<ApiEnvelope<SubjectInsight[]>>('/insights'))
}

export async function fetchInsight(id: number): Promise<SubjectInsight> {
  return requestData(api.get<ApiEnvelope<SubjectInsight>>(`/insights/${id}`))
}

export async function fetchTopics(): Promise<Topic[]> {
  return requestData(api.get<ApiEnvelope<Topic[]>>('/topics'))
}

export async function fetchTopic(slug: string): Promise<TopicDetail> {
  return requestData(api.get<ApiEnvelope<TopicDetail>>(`/topics/${slug}`))
}

export async function fetchPostDetail(postId: number): Promise<{ post: Post; comments: Comment[] }> {
  return requestData(api.get<ApiEnvelope<{ post: Post; comments: Comment[] }>>(`/posts/${postId}`))
}

export async function createComment(postId: number, content: string): Promise<Comment> {
  return requestData(api.post<ApiEnvelope<Comment>>(`/posts/${postId}/comments`, { content }))
}

export async function reportPost(postId: number, reason: string, detail = ''): Promise<ContentReport> {
  return requestData(api.post<ApiEnvelope<ContentReport>>(`/posts/${postId}/report`, { reason, detail }))
}

export async function togglePostLike(postId: number): Promise<ToggleResult> {
  return requestData(api.post<ApiEnvelope<ToggleResult>>(`/posts/${postId}/like`))
}

export async function togglePostFavorite(postId: number): Promise<ToggleResult> {
  return requestData(api.post<ApiEnvelope<ToggleResult>>(`/posts/${postId}/favorite`))
}

export async function toggleFollowAuthor(authorName: string): Promise<{ active: boolean }> {
  return requestData(api.post<ApiEnvelope<{ active: boolean }>>(`/authors/${encodeURIComponent(authorName)}/follow`))
}

export interface ConversationPage {
  items: Conversation[]
  nextCursor?: string
  hasMore: boolean
}

export async function fetchConversations(query: { limit?: number; cursor?: string } = {}): Promise<ConversationPage> {
  const response = await api.get<ApiEnvelope<Conversation[]>>('/messages', { params: query })
  return { items: response.data.data, nextCursor: response.data.meta?.nextCursor, hasMore: response.data.meta?.hasMore ?? false }
}

export interface DirectMessagePage {
  items: DirectMessage[]
  nextCursor?: string
  hasMore: boolean
}

export async function fetchDirectMessages(peerName: string, query: { limit?: number; cursor?: string } = {}): Promise<DirectMessagePage> {
  const response = await api.get<ApiEnvelope<DirectMessage[]>>(`/messages/${encodeURIComponent(peerName)}`, { params: query })
  return { items: response.data.data, nextCursor: response.data.meta?.nextCursor, hasMore: response.data.meta?.hasMore ?? false }
}

export async function sendDirectMessage(recipientName: string, content: string): Promise<DirectMessage> {
  return requestData(api.post<ApiEnvelope<DirectMessage>>('/messages', { recipientName, content }))
}

export async function requestChoiceAdvice(profile: ChoiceProfile, question = ''): Promise<ChoiceAdvice> {
  return requestData(api.post<ApiEnvelope<ChoiceAdvice>>('/ai/choice-advice', { profile, question }))
}
