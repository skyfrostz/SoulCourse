import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed } from 'vue'
import { apiDataEnabled, createComment, fetchPostDetail } from '../lib/api'
import { sampleComments } from '../lib/sampleData'
import { useForumStore } from '../stores/forum'

export function usePostComments(postId: () => number) {
  const queryClient = useQueryClient()
  const forumStore = useForumStore()

  const detailQuery = useQuery({
    queryKey: computed(() => ['post-detail', postId()]),
    queryFn: () => fetchPostDetail(postId()),
    enabled: apiDataEnabled,
  })

  const comments = computed(() => {
    const id = postId()
    const fallback = detailQuery.data.value?.comments ?? sampleComments[id] ?? []
    return forumStore.getPostComments(id, fallback)
  })

  const submitMutation = useMutation({
    mutationFn: async (content: string) => {
      if (!apiDataEnabled) return forumStore.addLocalComment(postId(), content)
      try {
        return await createComment(postId(), content)
      } catch {
        return forumStore.addLocalComment(postId(), content)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['post-detail', postId()] })
      queryClient.invalidateQueries({ queryKey: ['posts'] })
    },
  })

  return {
    comments,
    submitComment: submitMutation.mutateAsync,
    isSubmitting: submitMutation.isPending,
  }
}
