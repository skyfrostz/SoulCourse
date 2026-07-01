import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed } from 'vue'
import { createComment, fetchPostDetail } from '../lib/api'
import { sampleComments } from '../lib/sampleData'

export function usePostComments(postId: () => number) {
  const queryClient = useQueryClient()

  const detailQuery = useQuery({
    queryKey: computed(() => ['post-detail', postId()]),
    queryFn: () => fetchPostDetail(postId()),
  })

  const comments = computed(() => {
    const id = postId()
    return detailQuery.data.value?.comments ?? sampleComments[id] ?? []
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
