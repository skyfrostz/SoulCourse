<script setup lang="ts">
import { BarChart3, Eye, MessageSquare, ThumbsUp, X } from '@lucide/vue'
import { categoryLabels } from '../lib/labels'
import { useForumStore } from '../stores/forum'

const forumStore = useForumStore()
</script>

<template>
  <aside class="discussion-drawer detail-drawer">
    <div class="drawer-header">
      <span class="category-chip data">{{ forumStore.detailPanel.kind === 'topic' ? '热门话题' : '精选建议' }}</span>
      <button class="icon-button" aria-label="关闭详情" @click="forumStore.closeDetail">
        <X :size="18" />
      </button>
    </div>

    <section v-if="forumStore.detailPanel.kind === 'topic'" class="drawer-post">
      <h1># {{ forumStore.detailPanel.detail.topic.title }}</h1>
      <p>{{ forumStore.detailPanel.detail.topic.summary }}</p>
      <div class="drawer-actions">
        <button><Eye :size="17" /> {{ forumStore.detailPanel.detail.topic.viewsCount }}</button>
        <button><MessageSquare :size="17" /> {{ forumStore.detailPanel.detail.topic.postsCount }}</button>
      </div>

      <div class="related-post-list">
        <button
          v-for="post in forumStore.detailPanel.detail.posts"
          :key="post.id"
          class="related-post"
          @click="forumStore.selectPost(post.id)"
        >
          <span class="category-chip" :class="post.category">{{ categoryLabels[post.category] }}</span>
          <strong>{{ post.title }}</strong>
          <small><MessageSquare :size="14" /> {{ post.commentsCount }} <ThumbsUp :size="14" /> {{ post.likesCount }}</small>
        </button>
      </div>
    </section>

    <section v-else-if="forumStore.detailPanel.kind === 'insight'" class="drawer-post">
      <h1>{{ forumStore.detailPanel.detail.combination }}</h1>
      <div class="insight-score">
        <span>
          <strong>{{ forumStore.detailPanel.detail.heat }}</strong>
          热度
        </span>
        <span>
          <strong>{{ forumStore.detailPanel.detail.matchRate }}%</strong>
          匹配度
        </span>
      </div>
      <p>{{ forumStore.detailPanel.detail.advice }}</p>
      <p>{{ forumStore.detailPanel.detail.details }}</p>
      <div class="advice-card-large">
        <BarChart3 :size="22" />
        <span>
          <strong>{{ forumStore.detailPanel.detail.trend }}</strong>
          <small>更新时间：{{ new Date(forumStore.detailPanel.detail.updatedAt).toLocaleDateString('zh-CN') }}</small>
        </span>
      </div>
    </section>
  </aside>
</template>
