<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { Bookmark, ExternalLink, MessageSquare, ThumbsUp } from '@lucide/vue'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { togglePostFavorite, togglePostLike } from '../lib/api'
import { categoryLabels, roleLabels, subjectLabels, trackLabels } from '../lib/labels'
import { appAssetUrl } from '../lib/runtime'
import { useForumStore } from '../stores/forum'
import type { Post } from '../types/forum'

const props = defineProps<{
  post: Post
}>()

const forumStore = useForumStore()
const route = useRoute()
const router = useRouter()
const queryClient = useQueryClient()
const livePost = computed(() => forumStore.hydratePost(props.post))

const likeMutation = useMutation({
  mutationFn: () => togglePostLike(props.post.id),
  onSuccess: () => queryClient.invalidateQueries({ queryKey: ['posts'] }),
  onError: () => {
    forumStore.refreshHint = '点赞失败，请检查登录状态或稍后重试。'
  },
})

const favoriteMutation = useMutation({
  mutationFn: () => togglePostFavorite(props.post.id),
  onSuccess: () => queryClient.invalidateQueries({ queryKey: ['posts'] }),
  onError: () => {
    forumStore.refreshHint = '收藏失败，请检查登录状态或稍后重试。'
  },
})

function formatCount(value: number) {
  if (value >= 1000) return `${(value / 1000).toFixed(1)}k`
  return String(value)
}

function toggleLike() {
  if (likeMutation.isPending.value) return
  if (!forumStore.requireAuth()) return
  likeMutation.mutate()
}

function toggleFavorite() {
  if (favoriteMutation.isPending.value) return
  if (!forumStore.requireAuth()) return
  favoriteMutation.mutate()
}

function openPost(event: MouseEvent, targetSection?: 'comments') {
  if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
  const opensInFeedModal = route.name === 'home' || route.name === 'community'
  if (!opensInFeedModal) {
    event.preventDefault()
    void router.push(`/posts/${livePost.value.id}`)
    return
  }

  event.preventDefault()
  const query: Record<string, string | string[]> = { ...route.query, post: String(livePost.value.id) }
  if (targetSection) query.targetSection = targetSection
  else delete query.targetSection
  void router.push({ name: 'home', query })
}
</script>

<template>
  <article class="post-card forum-row-card">
      <a v-if="livePost.imageUrls?.length" class="post-image-strip" :href="`/posts/${livePost.id}`" @click="openPost">
        <img :src="appAssetUrl(livePost.imageUrls[0])" :alt="livePost.title" />
        <span v-if="livePost.imageUrls.length > 1">{{ livePost.imageUrls.length }} 图</span>
      </a>
    <a v-else class="post-title-cover" :class="`cover-${livePost.track}`" :href="`/posts/${livePost.id}`" @click="openPost">
      <span>{{ trackLabels[livePost.track] }}</span>
      <strong>{{ livePost.title }}</strong>
      <small>{{ livePost.electives.map((item) => subjectLabels[item]).join(' · ') }}</small>
    </a>

    <div class="post-card-content">
      <div class="post-meta-line">
        <span class="category-chip" :class="livePost.category">{{ categoryLabels[livePost.category] }}</span>
        <span>{{ trackLabels[livePost.track] }}</span>
      </div>

      <a class="post-hit-area" :href="`/posts/${livePost.id}`" @click="openPost">
        <h2>{{ livePost.title }}</h2>
        <p>{{ livePost.content }}</p>
      </a>

      <header class="post-card-head">
        <a v-if="livePost.sourcePlatform" class="author-profile-link source-author-link" :href="livePost.sourceUrl" target="_blank" rel="noreferrer">
          <img v-if="livePost.sourceAvatarUrl" class="small-avatar source-avatar" :src="appAssetUrl(livePost.sourceAvatarUrl)" :alt="livePost.sourceAuthor || livePost.authorName" />
          <span v-else class="small-avatar">{{ (livePost.sourceAuthor || livePost.authorName).slice(0, 1) }}</span>
          <span>
            <strong>{{ livePost.sourceAuthor || livePost.authorName }} <em class="source-badge">小红书来源</em></strong>
            <small>原作者 · 查看原文 <ExternalLink :size="12" /></small>
          </span>
        </a>
        <RouterLink v-else class="author-profile-link" :to="`/users/${encodeURIComponent(livePost.authorName)}`">
          <span class="small-avatar">{{ livePost.authorName.slice(0, 1) }}</span>
          <span>
            <strong>
              {{ livePost.authorName }}
              <em v-if="['teacher', 'counselor'].includes(livePost.authorRole)" class="verified-badge">认证</em>
            </strong>
            <small>{{ livePost.grade }} · {{ roleLabels[livePost.authorRole] }} · {{ livePost.electives.map((item) => subjectLabels[item]).join('') }}</small>
          </span>
        </RouterLink>
      </header>

      <div v-if="livePost.tags?.length" class="tag-row post-card-tags" aria-label="帖子标签">
        <span v-for="tag in livePost.tags" :key="tag"># {{ tag }}</span>
      </div>
    </div>

    <footer class="post-card-actions">
      <a :href="`/posts/${livePost.id}`" @click="openPost($event, 'comments')"><MessageSquare :size="16" /> {{ formatCount(livePost.commentsCount) }} 条讨论</a>
      <div class="post-stats">
        <button type="button" :class="{ active: livePost.viewerLiked }" :aria-pressed="livePost.viewerLiked" :disabled="likeMutation.isPending.value" aria-label="点赞帖子" @click.stop="toggleLike">
          <ThumbsUp :size="16" /> {{ formatCount(livePost.likesCount) }}
        </button>
        <button type="button" :class="{ active: livePost.viewerFavorited }" :aria-pressed="livePost.viewerFavorited" :disabled="favoriteMutation.isPending.value" aria-label="收藏帖子" @click.stop="toggleFavorite">
          <Bookmark :size="16" /> {{ formatCount(livePost.favoritesCount) }}
        </button>
      </div>
    </footer>
  </article>
</template>
