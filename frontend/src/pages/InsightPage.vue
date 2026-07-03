<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { BarChart3, ChevronLeft, Gauge, TrendingUp } from '@lucide/vue'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PostCard from '../components/PostCard.vue'
import { apiDataEnabled, fetchInsight } from '../lib/api'
import { samplePosts } from '../lib/sampleData'
import { useForumStore } from '../stores/forum'

const route = useRoute()
const router = useRouter()
const forumStore = useForumStore()
const insightId = computed(() => Number(route.params.id))
const insightQuery = useQuery({
  queryKey: computed(() => ['insight-detail', insightId.value]),
  queryFn: () => fetchInsight(insightId.value),
  enabled: apiDataEnabled,
})
const insight = computed(() => insightQuery.data.value)
const relatedPosts = computed(() => {
  const current = insight.value
  if (!current) return []
  return samplePosts.filter((post) => post.tags.includes(current.combination) || post.content.includes(current.combination)).slice(0, 6)
})
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <article v-if="insight" class="insight-article">
      <div class="breadcrumb">首页 / 数据建议 / 选科组合趋势</div>
      <h1>{{ insight.combination }}</h1>
      <p class="article-lead">{{ insight.advice }}</p>
      <div class="insight-score insight-score-wide">
        <span><TrendingUp :size="18" /><strong>{{ insight.heat }}</strong> 热度</span>
        <span><Gauge :size="18" /><strong>{{ insight.matchRate }}%</strong> 匹配度</span>
        <span><BarChart3 :size="18" /><strong>{{ insight.trend }}</strong> 趋势</span>
      </div>
      <p class="article-body">{{ insight.details }}</p>
      <div class="publish-strip">
        <span>想结合你的成绩和目标专业继续讨论？</span>
        <button class="primary-wide" @click="forumStore.openPublish('question')">发布提问</button>
      </div>
    </article>

    <section v-else-if="!insightQuery.isLoading.value" class="empty-state detail-empty-state">
      <h1>趋势数据暂时无法加载</h1>
      <p>请返回趋势中心刷新，或稍后重试。</p>
    </section>

    <section class="feed-panel topic-feed-panel">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">同组合笔记</button>
        </div>
      </div>
      <div class="feed-grid">
        <PostCard v-for="post in relatedPosts" :key="post.id" :post="post" />
      </div>
    </section>
  </main>
</template>
