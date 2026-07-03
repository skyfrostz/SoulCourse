import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed } from 'vue'
import { apiDataEnabled, createComment, fetchPostDetail } from '../lib/api'

export function usePostComments(postId: () => number) {
  const queryClient = useQueryClient()

  const detailQuery = useQuery({
    queryKey: computed(() => ['post-detail', postId()]),
    queryFn: () => fetchPostDetail(postId()),
    enabled: apiDataEnabled,
  })

  const comments = computed(() => {
    return detailQuery.data.value?.comments ?? []
  })

  const submitMutation = useMutation({
    mutationFn: (content: string) => createComment(postId(), content),
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
