<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { Bookmark, MessageSquare, Send, Share2, ThumbsUp, X } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import { toggleFollowAuthor, togglePostFavorite, togglePostLike } from '../lib/api'
import { categoryLabels, roleLabels, subjectLabels } from '../lib/labels'
import { usePostComments } from '../composables/usePostComments'
import { useForumStore } from '../stores/forum'
import type { Post } from '../types/forum'

const props = defineProps<{
  post: Post
}>()

const forumStore = useForumStore()
const queryClient = useQueryClient()
const draft = ref('')
const livePost = ref<Post>(props.post)
const { comments, submitComment, isSubmitting } = usePostComments(() => props.post.id)

watch(
  () => props.post,
  (post) => {
    livePost.value = post
    draft.value = ''
  },
  { immediate: true },
)

const likeMutation = useMutation({
  mutationFn: () => togglePostLike(props.post.id),
  onSuccess: (result) => {
    livePost.value = { ...livePost.value, viewerLiked: result.active, likesCount: result.count }
    queryClient.invalidateQueries({ queryKey: ['posts'] })
  },
})

const favoriteMutation = useMutation({
  mutationFn: () => togglePostFavorite(props.post.id),
  onSuccess: (result) => {
    livePost.value = { ...livePost.value, viewerFavorited: result.active, favoritesCount: result.count }
    queryClient.invalidateQueries({ queryKey: ['posts'] })
  },
})

const followMutation = useMutation({
  mutationFn: () => toggleFollowAuthor(props.post.authorName),
  onSuccess: (result) => {
    livePost.value = { ...livePost.value, viewerFollowing: result.active }
    queryClient.invalidateQueries({ queryKey: ['posts'] })
  },
})

const commentCount = computed(() => comments.value.length)

async function submit() {
  if (!forumStore.requireAuth()) return
  const content = draft.value.trim()
  if (!content) return
  await submitComment(content)
  livePost.value = { ...livePost.value, commentsCount: livePost.value.commentsCount + 1 }
  draft.value = ''
}

function like() {
  if (forumStore.requireAuth()) likeMutation.mutate()
}

function favorite() {
  if (forumStore.requireAuth()) favoriteMutation.mutate()
}

function follow() {
  if (forumStore.requireAuth()) followMutation.mutate()
}
</script>

<template>
  <aside class="discussion-drawer">
    <div class="drawer-header">
      <span class="category-chip" :class="livePost.category">{{ categoryLabels[livePost.category] }}</span>
      <button class="icon-button" aria-label="关闭详情">
        <X :size="18" />
      </button>
    </div>

    <article class="drawer-post">
      <h1>{{ livePost.title }}</h1>
      <div class="drawer-author">
        <span class="avatar medium">{{ livePost.authorName.slice(0, 1) }}</span>
        <span>
          <strong>{{ livePost.authorName }}</strong>
          <small>{{ livePost.grade }} · {{ roleLabels[livePost.authorRole] }}</small>
        </span>
        <button class="follow-button" :class="{ active: livePost.viewerFollowing }" @click="follow">
          {{ livePost.viewerFollowing ? '已关注' : '+ 关注' }}
        </button>
      </div>

      <p>{{ livePost.content }}</p>
      <p>
        组合：{{ livePost.track === 'physics' ? '物理方向' : '历史方向' }} ·
        {{ livePost.electives.map((item) => subjectLabels[item]).join(' + ') }}
      </p>

      <div class="drawer-actions">
        <button><Share2 :size="17" /> 分享</button>
        <button><MessageSquare :size="17" /> {{ livePost.commentsCount }}</button>
        <button :class="{ liked: livePost.viewerLiked }" @click="like">
          <ThumbsUp :size="17" /> {{ livePost.likesCount }}
        </button>
        <button :class="{ liked: livePost.viewerFavorited }" @click="favorite">
          <Bookmark :size="17" /> {{ livePost.viewerFavorited ? '已收藏' : '收藏' }}
        </button>
      </div>
    </article>

    <section class="comment-section">
      <div class="comment-title-row">
        <h2>评论 {{ commentCount }}</h2>
        <button>按时间</button>
      </div>

      <form class="comment-form" @submit.prevent="submit">
        <input v-model="draft" type="text" :placeholder="forumStore.isAuthed ? '发表评论' : '登录后发表评论'" />
        <button :disabled="!draft.trim() || isSubmitting" type="submit">
          <Send :size="16" />
          发表评论
        </button>
      </form>

      <div class="comment-list">
        <article v-for="comment in comments" :key="comment.id" class="comment-item">
          <span class="small-avatar">{{ comment.author.slice(0, 1) }}</span>
          <div>
            <div class="comment-meta">
              <strong>{{ comment.author }}</strong>
              <span>{{ roleLabels[comment.role] }}</span>
            </div>
            <p>{{ comment.content }}</p>
            <div class="comment-actions">
              <span>{{ new Date(comment.createdAt).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) }}</span>
              <button>回复</button>
            </div>
          </div>
        </article>
      </div>
    </section>
  </aside>
</template>
