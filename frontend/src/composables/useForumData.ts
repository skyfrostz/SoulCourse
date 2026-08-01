import { useQuery } from '@tanstack/vue-query'
import { computed, ref, watch } from 'vue'
import { apiDataEnabled, fetchFeedPage, fetchInsights, fetchTopics } from '../lib/api'
import { useForumStore } from '../stores/forum'
import type { Post } from '../types/forum'

export function useForumData() {
  const forumStore = useForumStore()
  const pageCursors = ref<Record<number, string | undefined>>({ 1: undefined })
  const feedIdentity = computed(() => JSON.stringify({
    track: forumStore.filter.track,
    subjects: forumStore.filter.subjects,
    category: forumStore.filter.category,
    keyword: forumStore.filter.keyword,
    sort: forumStore.filter.sort,
  }))

  watch(feedIdentity, () => {
    pageCursors.value = { 1: undefined }
  })

  const postsQuery = useQuery({
    queryKey: computed(() => [
      'posts',
      forumStore.filter.track,
      forumStore.filter.subjects.join(','),
      forumStore.filter.category,
      forumStore.filter.keyword,
      forumStore.filter.sort,
      forumStore.page,
      forumStore.session?.user.id ?? 'guest',
    ]),
    queryFn: async () => {
      const page = forumStore.page
      const result = await fetchFeedPage(
        forumStore.filter,
        page,
        forumStore.pageSize,
        pageCursors.value[page],
      )
      if (result.hasMore && result.nextCursor) {
        pageCursors.value[page + 1] = result.nextCursor
      } else {
        delete pageCursors.value[page + 1]
      }
      return result
    },
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
    const apiPosts = postsQuery.data.value?.items ?? []
    const merged = new Map<number, Post>()
    apiPosts.forEach((post) => {
      merged.set(post.id, forumStore.hydratePost(post))
    })
    return Array.from(merged.values())
  })

  return {
    posts,
    hasMore: computed(() => postsQuery.data.value?.hasMore ?? false),
    insights: computed(() => insightsQuery.data.value ?? []),
    topics: computed(() => topicsQuery.data.value ?? []),
    source: computed(() => 'api' as const),
    isLoading: computed(() => postsQuery.isLoading.value && !postsQuery.data.value),
    hasError: computed(() => postsQuery.isError.value),
    observationLoading: computed(() =>
      (insightsQuery.isLoading.value && !insightsQuery.data.value) ||
      (topicsQuery.isLoading.value && !topicsQuery.data.value),
    ),
    observationHasError: computed(() => insightsQuery.isError.value || topicsQuery.isError.value),
    refetchObservations: () => {
      void insightsQuery.refetch()
      void topicsQuery.refetch()
    },
    refetch: () => {
      void postsQuery.refetch()
      void insightsQuery.refetch()
      void topicsQuery.refetch()
    },
  }
}
