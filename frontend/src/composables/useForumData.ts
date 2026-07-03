import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { apiDataEnabled, fetchInsights, fetchPosts, fetchTopics } from '../lib/api'
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

  const posts = computed(() => {
    const apiPosts = postsQuery.data.value ?? []
    const merged = new Map<number, Post>()
    apiPosts.forEach((post) => {
      merged.set(post.id, forumStore.hydratePost(post))
    })
    return Array.from(merged.values())
  })

  return {
    posts,
    insights: computed(() => insightsQuery.data.value ?? []),
    topics: computed(() => topicsQuery.data.value ?? []),
    source: computed(() => 'api' as const),
    isLoading: computed(() => postsQuery.isLoading.value && !postsQuery.data.value),
    hasError: computed(() => postsQuery.isError.value || insightsQuery.isError.value || topicsQuery.isError.value),
  }
}
