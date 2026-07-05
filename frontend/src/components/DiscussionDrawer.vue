<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { Bookmark, MessageSquare, Send, Share2, ThumbsUp, X } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import { toggleFollowAuthor, togglePostFavorite, togglePostLike } from '../lib/api'
import { categoryLabels, roleLabels, subjectLabels } from '../lib/labels'
import { buildAppUrl } from '../lib/runtime'
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
    livePost.value = forumStore.hydratePost(post)
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
    forumStore.setUserFollow({
      name: props.post.authorName,
      role: props.post.authorRole,
      province: props.post.province,
      grade: props.post.grade,
      followedAt: new Date().toISOString(),
    }, result.active)
    queryClient.invalidateQueries({ queryKey: ['posts'] })
  },
})

const commentCount = computed(() => comments.value.length)

function closeDrawer() {
  forumStore.closeDetail()
}

async function submit() {
  if (!forumStore.requireAuth()) return
  const content = draft.value.trim()
  if (!content) return
  await submitComment(content)
  livePost.value = { ...livePost.value, commentsCount: commentCount.value }
  draft.value = ''
}

function like() {
  if (!forumStore.requireAuth()) return
  likeMutation.mutate(undefined, {
    onError: () => {
      forumStore.refreshHint = '点赞失败，请稍后重试。'
    },
  })
}

function favorite() {
  if (!forumStore.requireAuth()) return
  favoriteMutation.mutate(undefined, {
    onError: () => {
      forumStore.refreshHint = '收藏失败，请稍后重试。'
    },
  })
}

function follow() {
  if (!forumStore.requireAuth()) return
  followMutation.mutate(undefined, {
    onError: () => {
      forumStore.refreshHint = '关注失败，请稍后重试。'
    },
  })
}

async function sharePost() {
  const url = buildAppUrl(`/posts/${livePost.value.id}`)
  if (navigator.share) {
    await navigator.share({ title: livePost.value.title, text: livePost.value.content, url }).catch(() => undefined)
    return
  }
  await navigator.clipboard?.writeText(url).catch(() => undefined)
  forumStore.refreshHint = '帖子链接已复制，可以发给同学一起讨论。'
  window.setTimeout(() => {
    forumStore.refreshHint = ''
  }, 1600)
}

function replyTo(author: string) {
  if (!forumStore.requireAuth()) return
  draft.value = `@${author} `
}
</script>

<template>
  <aside class="discussion-drawer">
    <div class="drawer-header">
      <span class="category-chip" :class="livePost.category">{{ categoryLabels[livePost.category] }}</span>
      <button class="icon-button" aria-label="关闭详情" @click="closeDrawer">
        <X :size="18" />
      </button>
    </div>

    <article class="drawer-post">
      <h1>{{ livePost.title }}</h1>
      <div class="drawer-author">
        <RouterLink class="avatar medium user-link-avatar" :to="`/users/${encodeURIComponent(livePost.authorName)}`">
          {{ livePost.authorName.slice(0, 1) }}
        </RouterLink>
        <span>
          <RouterLink class="author-name-link" :to="`/users/${encodeURIComponent(livePost.authorName)}`">
            <strong>{{ livePost.authorName }}</strong>
          </RouterLink>
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
        <button @click="sharePost"><Share2 :size="17" /> 分享</button>
        <button><MessageSquare :size="17" /> {{ commentCount }}</button>
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
        <span class="comment-sort-label">按时间</span>
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
          <RouterLink class="small-avatar user-link-avatar" :to="`/users/${encodeURIComponent(comment.author)}`">
            {{ comment.author.slice(0, 1) }}
          </RouterLink>
          <div>
            <div class="comment-meta">
              <RouterLink :to="`/users/${encodeURIComponent(comment.author)}`">
                <strong>{{ comment.author }}</strong>
              </RouterLink>
              <span>{{ roleLabels[comment.role] }}</span>
            </div>
            <p>{{ comment.content }}</p>
            <div class="comment-actions">
              <span>{{ new Date(comment.createdAt).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) }}</span>
              <button @click="replyTo(comment.author)">回复</button>
            </div>
          </div>
        </article>
      </div>
    </section>
  </aside>
</template>
