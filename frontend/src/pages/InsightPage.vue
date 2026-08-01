<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { BarChart3, ChevronLeft, Gauge, TrendingUp } from '@lucide/vue'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PostCard from '../components/PostCard.vue'
import { apiDataEnabled, fetchInsight, fetchPostCollection } from '../lib/api'
import { useForumStore } from '../stores/forum'
import { useOnlineState } from '../composables/useOnlineState'

const route = useRoute()
const router = useRouter()
const forumStore = useForumStore()
const { isOffline } = useOnlineState()
const insightId = computed(() => Number(route.params.id))
const insightQuery = useQuery({
  queryKey: computed(() => ['insight-detail', insightId.value]),
  queryFn: () => fetchInsight(insightId.value),
  enabled: apiDataEnabled,
})
const insight = computed(() => insightQuery.data.value)
const subjectTag = computed(() => combinationTag(insight.value?.combination ?? ''))
const relatedPostsQuery = useQuery({
  queryKey: computed(() => ['insight-posts', subjectTag.value]),
  queryFn: () => fetchPostCollection({ tag: subjectTag.value, sort: 'latest', limit: 50 }),
  enabled: computed(() => Boolean(subjectTag.value)),
})
const relatedPosts = computed(() => relatedPostsQuery.data.value ?? [])

function combinationTag(combination: string) {
  const abbreviations: Record<string, string> = { 物理: '物', 历史: '史', 化学: '化', 生物: '生', 政治: '政', 地理: '地' }
  return combination.split('+').map((item) => abbreviations[item.trim()] ?? '').join('')
}
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <article v-if="insight" class="insight-article">
      <div class="breadcrumb">首页 / 数据建议 / 选科组合趋势</div>
      <h1>{{ insight.combination }}</h1>
      <p class="article-lead">{{ insight.advice }}</p>
      <div class="insight-score insight-score-wide">
        <span><TrendingUp :size="18" /><strong>{{ insight.heat }}</strong> {{ insight.unit }}</span>
        <span><Gauge :size="18" /><strong>{{ insight.matchRate }}%</strong> 数据集占比</span>
        <span><BarChart3 :size="18" /><strong>{{ insight.trend }}</strong> 选科要求</span>
      </div>
      <p class="article-body">{{ insight.details }}</p>
      <p class="article-body">统计范围：{{ insight.scope }}。{{ insight.methodology }}</p>
      <a class="note-source-link advice-source-large" :href="insight.sourceUrl" target="_blank" rel="noreferrer">
        来源：{{ insight.sourceName }} · 抓取于 {{ new Date(insight.capturedAt).toLocaleDateString('zh-CN') }}
      </a>
      <div class="publish-strip">
        <span>想结合你的成绩和目标专业继续讨论？</span>
        <button class="primary-wide" @click="forumStore.openPublish('question')">发布提问</button>
      </div>
    </article>

    <section v-else-if="insightQuery.isError.value || isOffline" class="empty-state detail-empty-state">
      <h1>{{ isOffline ? '当前网络不可用' : '趋势数据暂时无法加载' }}</h1>
      <p>{{ isOffline ? '恢复网络后再重试，页面不会生成模拟趋势。' : '请返回趋势中心刷新，或稍后重试。' }}</p>
      <button class="primary-wide compact" type="button" @click="insightQuery.refetch()">重新加载</button>
    </section>

    <section v-else-if="insightQuery.isLoading.value" class="empty-state detail-empty-state" aria-live="polite">
      <p>正在加载趋势详情...</p>
    </section>

    <section v-if="insight" class="feed-panel topic-feed-panel">
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
