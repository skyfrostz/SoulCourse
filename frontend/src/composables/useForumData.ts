import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { apiDataEnabled, fetchInsights, fetchPosts, fetchTopics } from '../lib/api'
import { sampleComments, sampleInsights, samplePosts, sampleTopics } from '../lib/sampleData'
import { useForumStore } from '../stores/forum'
import type { Post } from '../types/forum'

export function useForumData() {
  const forumStore = useForumStore()

  const postsQuery = useQuery({
    queryKey: computed(() => ['posts', forumStore.filter, forumStore.page, forumStore.session?.user.id ?? 'guest']),
    queryFn: () => fetchPosts(forumStore.filter, forumStore.page, forumStore.pageSize),
    enabled: apiDataEnabled,
  })

  const insightsQuery = useQuery({
    queryKey: ['insights'],
    queryFn: fetchInsights,
    enabled: apiDataEnabled,
  })

  const topicsQuery = useQuery({
    queryKey: ['topics'],
    queryFn: fetchTopics,
    enabled: apiDataEnabled,
  })

  const localPosts = computed(() => {
    const keyword = forumStore.filter.keyword.trim()
    const sourcePosts = [...forumStore.getCreatedPosts(), ...samplePosts]
    const filtered = sourcePosts.filter((post) => {
      const categoryFocused = forumStore.filter.category !== 'all'
      const matchesTrack = categoryFocused || post.track === forumStore.filter.track
      const matchesSubjects = categoryFocused || forumStore.filter.subjects.every((subject) => post.electives.includes(subject))
      const matchesCategory =
        forumStore.filter.category === 'all' || post.category === forumStore.filter.category
      const matchesKeyword =
        !keyword ||
        [
          post.title,
          post.content,
          post.authorName,
          post.province,
          post.grade,
          post.category,
          post.track,
          ...post.tags,
          ...post.electives,
        ].some((value) => value.includes(keyword))
      return matchesTrack && matchesSubjects && matchesCategory && matchesKeyword
    })
    const sorted = [...filtered].sort((a, b) => {
      if (forumStore.filter.sort === 'latest') {
        return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      }
      if (forumStore.filter.sort === 'hot') {
        return b.likesCount + b.commentsCount * 4 - (a.likesCount + a.commentsCount * 4)
      }
      const score = (post: typeof a) =>
        Math.min(post.likesCount, 300) * 0.8 +
        Math.min(post.commentsCount, 80) * 4 +
        Math.min(post.favoritesCount, 120) * 3 +
        (['teacher', 'counselor'].includes(post.authorRole) ? 45 : 0) +
        (post.likesCount < 150 ? 65 : 0)
      return score(b) - score(a)
    })
    const start = (forumStore.page - 1) * forumStore.pageSize
    return sorted.slice(start, start + forumStore.pageSize)
  })

  const posts = computed(() => {
    const apiPosts = postsQuery.data.value ?? []
    const merged = new Map<number, Post>()
    ;[...localPosts.value, ...apiPosts].forEach((post) => {
      const hydrated = forumStore.hydratePost(post)
      const fallbackComments = sampleComments[post.id]
      merged.set(post.id, {
        ...hydrated,
        commentsCount: fallbackComments
          ? forumStore.getActualCommentCount(post.id, fallbackComments)
          : (forumStore.localEngagement.comments[post.id]?.length ?? hydrated.commentsCount),
      })
    })
    return Array.from(merged.values())
  })
  const insights = computed(() => {
    const apiInsights = insightsQuery.data.value ?? []
    return [...apiInsights, ...sampleInsights.filter((item) => !apiInsights.some((apiItem) => apiItem.id === item.id))]
  })
  const source = computed(() => (postsQuery.data.value ? 'api' : 'local'))

  return {
    posts,
    insights,
    topics: computed(() => (topicsQuery.data.value?.length ? topicsQuery.data.value : sampleTopics)),
    source,
    isLoading: computed(() => postsQuery.isLoading.value && !localPosts.value.length && !postsQuery.data.value),
  }
}
