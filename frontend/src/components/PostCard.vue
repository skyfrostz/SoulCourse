<script setup lang="ts">
import { Bookmark, MessageSquare, ThumbsUp } from '@lucide/vue'
import { categoryLabels, roleLabels, subjectLabels, trackLabels } from '../lib/labels'
import type { Post } from '../types/forum'

defineProps<{
  post: Post
}>()

function formatCount(value: number) {
  if (value >= 1000) return `${(value / 1000).toFixed(1)}k`
  return String(value)
}
</script>

<template>
  <article class="post-card forum-row-card">
    <RouterLink class="post-hit-area" :to="`/posts/${post.id}`">
      <div class="post-meta-line">
        <span class="category-chip" :class="post.category">{{ categoryLabels[post.category] }}</span>
        <span>{{ trackLabels[post.track] }}</span>
      </div>

      <h2>{{ post.title }}</h2>
      <p>{{ post.content }}</p>

      <div v-if="post.imageUrls?.length" class="post-image-strip">
        <img :src="post.imageUrls[0]" :alt="post.title" />
        <span v-if="post.imageUrls.length > 1">{{ post.imageUrls.length }} 图</span>
      </div>

      <div v-if="post.tags?.length" class="tag-row">
        <span v-for="tag in post.tags.slice(0, 4)" :key="tag"># {{ tag }}</span>
      </div>

      <div v-if="post.category === 'data'" class="mini-chart" aria-hidden="true">
        <span></span>
        <span></span>
        <span></span>
        <span></span>
      </div>

      <div v-if="post.id === 6" class="study-photo" aria-hidden="true"></div>

      <div class="post-footer">
        <div class="author-line">
          <span class="small-avatar">{{ post.authorName.slice(0, 1) }}</span>
          <span>
            <strong>
              {{ post.authorName }}
              <em v-if="['teacher', 'counselor'].includes(post.authorRole)" class="verified-badge">认证</em>
            </strong>
            <small>{{ post.grade }} · {{ roleLabels[post.authorRole] }} · {{ post.electives.map((item) => subjectLabels[item]).join('') }}</small>
          </span>
        </div>

        <div class="post-stats">
          <span><MessageSquare :size="15" /> {{ formatCount(post.commentsCount) }}</span>
          <span><ThumbsUp :size="15" /> {{ formatCount(post.likesCount) }}</span>
          <span><Bookmark :size="15" /> {{ formatCount(post.favoritesCount) }}</span>
        </div>
      </div>
    </RouterLink>
  </article>
</template>
