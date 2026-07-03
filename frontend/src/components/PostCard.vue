<script setup lang="ts">
import { Bookmark, MessageSquare, ThumbsUp } from '@lucide/vue'
import { computed } from 'vue'
import { categoryLabels, roleLabels, subjectLabels, trackLabels } from '../lib/labels'
import { sampleComments } from '../lib/sampleData'
import { useForumStore } from '../stores/forum'
import type { Post } from '../types/forum'

const props = defineProps<{
  post: Post
}>()

const forumStore = useForumStore()
const livePost = computed(() => {
  const hydrated = forumStore.hydratePost(props.post)
  const fallbackComments = sampleComments[props.post.id]
  return {
    ...hydrated,
    commentsCount: fallbackComments
      ? forumStore.getActualCommentCount(props.post.id, fallbackComments)
      : (forumStore.localEngagement.comments[props.post.id]?.length ?? hydrated.commentsCount),
  }
})

function formatCount(value: number) {
  if (value >= 1000) return `${(value / 1000).toFixed(1)}k`
  return String(value)
}

function toggleLike() {
  if (!forumStore.requireAuth()) return
  forumStore.toggleLocalLike(livePost.value)
}

function toggleFavorite() {
  if (!forumStore.requireAuth()) return
  forumStore.toggleLocalFavorite(livePost.value)
}
</script>

<template>
  <article class="post-card forum-row-card">
    <RouterLink class="post-hit-area" :to="`/posts/${livePost.id}`">
      <div class="post-meta-line">
        <span class="category-chip" :class="livePost.category">{{ categoryLabels[livePost.category] }}</span>
        <span>{{ trackLabels[livePost.track] }}</span>
      </div>

      <h2>{{ livePost.title }}</h2>
      <p>{{ livePost.content }}</p>

      <div v-if="livePost.imageUrls?.length" class="post-image-strip">
        <img :src="livePost.imageUrls[0]" :alt="livePost.title" />
        <span v-if="livePost.imageUrls.length > 1">{{ livePost.imageUrls.length }} 图</span>
      </div>

      <div v-if="livePost.tags?.length" class="tag-row">
        <span v-for="tag in livePost.tags.slice(0, 4)" :key="tag"># {{ tag }}</span>
      </div>

      <div v-if="livePost.category === 'data'" class="mini-chart" aria-hidden="true">
        <span></span>
        <span></span>
        <span></span>
        <span></span>
      </div>

      <div v-if="livePost.id === 6" class="study-photo" aria-hidden="true"></div>

      <div class="post-footer">
        <div class="author-line">
          <span class="small-avatar">{{ livePost.authorName.slice(0, 1) }}</span>
          <span>
            <strong>
              {{ livePost.authorName }}
              <em v-if="['teacher', 'counselor'].includes(livePost.authorRole)" class="verified-badge">认证</em>
            </strong>
            <small>{{ livePost.grade }} · {{ roleLabels[livePost.authorRole] }} · {{ livePost.electives.map((item) => subjectLabels[item]).join('') }}</small>
          </span>
        </div>

        <div class="post-stats">
          <span><MessageSquare :size="15" /> {{ formatCount(livePost.commentsCount) }}</span>
          <button type="button" :class="{ active: livePost.viewerLiked }" @click.prevent.stop="toggleLike">
            <ThumbsUp :size="15" /> {{ formatCount(livePost.likesCount) }}
          </button>
          <button type="button" :class="{ active: livePost.viewerFavorited }" @click.prevent.stop="toggleFavorite">
            <Bookmark :size="15" /> {{ formatCount(livePost.favoritesCount) }}
          </button>
        </div>
      </div>
    </RouterLink>
  </article>
</template>
