import { majorRequirements, type MajorRequirement } from './majorRequirements'
import { sampleComments, samplePosts } from './sampleData'
import type { Comment, Post } from '../types/forum'

interface ForumStoreLike {
  hydratePost(post: Post): Post
  getActualCommentCount(postId: number, fallback?: Comment[]): number
}

export interface MajorForumStats {
  postCount: number
  likesCount: number
  commentsCount: number
  favoritesCount: number
  hotScore: number
}

export function majorForumPath(major: string) {
  return `/requirements/${encodeURIComponent(major)}`
}

export function findMajorRequirement(major: string): MajorRequirement | undefined {
  const decoded = decodeURIComponent(major)
  return majorRequirements.find((item) => normalizeText(item.major) === normalizeText(decoded))
}

export function getRelatedMajorPosts(major: string, posts: Post[] = samplePosts): Post[] {
  const query = normalizeText(major)
  const exactTagMatches = posts.filter((post) => post.tags.some((tag) => normalizeText(tag) === query))
  const matchedPosts = exactTagMatches.length
    ? exactTagMatches
    : posts.filter((post) =>
        [post.title, post.content, post.authorName, post.province, ...post.tags]
          .map(normalizeText)
          .some((value) => value.includes(query)),
      )
  return matchedPosts
    .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
}

export function hydrateMajorPosts(major: string, forumStore: ForumStoreLike, posts: Post[] = samplePosts): Post[] {
  return getRelatedMajorPosts(major, posts).map((post) => {
    const hydrated = forumStore.hydratePost(post)
    const fallbackComments = sampleComments[post.id]
    return {
      ...hydrated,
      commentsCount: fallbackComments
        ? forumStore.getActualCommentCount(post.id, fallbackComments)
        : Math.max(hydrated.commentsCount, forumStore.getActualCommentCount(post.id, [])),
    }
  })
}

export function getMajorForumStats(major: string, forumStore: ForumStoreLike, posts: Post[] = samplePosts): MajorForumStats {
  const relatedPosts = hydrateMajorPosts(major, forumStore, posts)
  const likesCount = relatedPosts.reduce((sum, post) => sum + post.likesCount, 0)
  const commentsCount = relatedPosts.reduce((sum, post) => sum + post.commentsCount, 0)
  const favoritesCount = relatedPosts.reduce((sum, post) => sum + post.favoritesCount, 0)
  return {
    postCount: relatedPosts.length,
    likesCount,
    commentsCount,
    favoritesCount,
    hotScore: likesCount + commentsCount * 4 + favoritesCount * 2,
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
