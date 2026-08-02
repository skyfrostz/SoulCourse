import type { RealDataRecord } from './api'
import type { Post } from '../types/forum'

interface ForumStoreLike {
  hydratePost(post: Post): Post
}

export interface MajorForumStats {
  postCount: number
  likesCount: number
  commentsCount: number
  favoritesCount: number
  hotScore: number
}

export interface MajorRequirementCard {
  major: string
  category: string
  requiredSubjects: string[]
  suggestedCombination: string
  risk: string
  source: string
  sourceUrl?: string
  noteType: '已复核要求' | '需逐校核对'
  coverageStatus: RealDataRecord['coverageStatus']
  methodology: string
}

export function majorForumPath(major: string) {
  return `/requirements/${encodeURIComponent(major)}`
}

export function findMajorRequirement(major: string, requirements: MajorRequirementCard[] = []): MajorRequirementCard | undefined {
  const decoded = decodeURIComponent(major)
  return requirements.find((item) => normalizeText(item.major) === normalizeText(decoded))
}

export function toMajorRequirementCard(record: RealDataRecord): MajorRequirementCard {
  return {
    major: record.title,
    category: String(record.tags?.[record.tags.length - 1] || record.type || '专业要求'),
    requiredSubjects: record.requiredSubjects?.length ? record.requiredSubjects : ['以官方目录为准'],
    suggestedCombination: record.requiredSubjects?.length ? record.requiredSubjects.join(' + ') : '按官方目录逐校核对',
    risk: record.summary || record.methodology || '暂无摘要，请打开官方来源逐项核对。',
    source: record.source?.name || '官方来源',
    sourceUrl: record.source?.url || record.url,
    noteType: record.coverageStatus === 'verified' ? '已复核要求' : '需逐校核对',
    coverageStatus: record.coverageStatus,
    methodology: record.methodology,
  }
}

export function getRelatedMajorPosts(major: string, posts: Post[] = [], context?: { subjects?: string[]; category?: string }) {
  const query = normalizeText(major)
  const terms = [major, ...(context?.subjects ?? []), context?.category ?? ''].filter(Boolean).map(normalizeText)
  const matchedPosts = posts.filter((post) => {
    const searchable = [post.title, post.content, post.authorName, post.province, ...post.tags].map(normalizeText)
    return searchable.some((value) => value.includes(query)) || terms.slice(1).some((term) => searchable.some((value) => value.includes(term)))
  })
  return matchedPosts
    .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
}

export function hydrateMajorPosts(major: string, forumStore: ForumStoreLike, posts: Post[] = [], context?: { subjects?: string[]; category?: string }): Post[] {
  return getRelatedMajorPosts(major, posts, context).map((post) => forumStore.hydratePost(post))
}

export function getMajorForumStats(major: string, forumStore: ForumStoreLike, posts: Post[] = [], context?: { subjects?: string[]; category?: string }): MajorForumStats {
  const relatedPosts = hydrateMajorPosts(major, forumStore, posts, context)
  const likesCount = relatedPosts.reduce((sum, post) => sum + post.likesCount, 0)
  const commentsCount = relatedPosts.reduce((sum, post) => sum + post.commentsCount, 0)
  const favoritesCount = relatedPosts.reduce((sum, post) => sum + post.favoritesCount, 0)
  return {
    postCount: relatedPosts.length,
    likesCount,
    commentsCount,
    favoritesCount,
    hotScore: likesCount + commentsCount + favoritesCount,
  }
}

export function formatCompactCount(value: number) {
  if (value >= 10000) return `${(value / 10000).toFixed(1)}w`
  if (value >= 1000) return `${(value / 1000).toFixed(1)}k`
  return String(value)
}

function normalizeText(value: string) {
  return value.replace(/\s+/g, '').toLowerCase()
}
