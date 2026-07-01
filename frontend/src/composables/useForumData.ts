import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { fetchInsights, fetchPosts, fetchTopics } from '../lib/api'
import { sampleInsights, samplePosts } from '../lib/sampleData'
import { useForumStore } from '../stores/forum'

export function useForumData() {
  const forumStore = useForumStore()

  const postsQuery = useQuery({
    queryKey: computed(() => ['posts', forumStore.filter, forumStore.page, forumStore.session?.user.id ?? 'guest']),
    queryFn: () => fetchPosts(forumStore.filter, forumStore.page, forumStore.pageSize),
  })

  const insightsQuery = useQuery({
    queryKey: ['insights'],
    queryFn: fetchInsights,
  })

  const topicsQuery = useQuery({
    queryKey: ['topics'],
    queryFn: fetchTopics,
  })

  const localPosts = computed(() => {
    const keyword = forumStore.filter.keyword.trim()
    const filtered = samplePosts.filter((post) => {
      const matchesTrack = post.track === forumStore.filter.track
      const matchesSubjects = forumStore.filter.subjects.every((subject) => post.electives.includes(subject))
      const matchesCategory =
        forumStore.filter.category === 'all' || post.category === forumStore.filter.category
      const matchesKeyword =
        !keyword ||
        post.title.includes(keyword) ||
        post.content.includes(keyword) ||
        post.authorName.includes(keyword)
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

  const posts = computed(() => postsQuery.data.value ?? localPosts.value)
  const insights = computed(() => insightsQuery.data.value ?? sampleInsights)
  const source = computed(() => (postsQuery.data.value ? 'api' : 'local'))

  return {
    posts,
    insights,
    topics: computed(() => topicsQuery.data.value ?? []),
    source,
    isLoading: computed(() => postsQuery.isLoading.value && !postsQuery.data.value),
  }
}
