<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { BarChart3, ChevronLeft, Gauge, TrendingUp } from '@lucide/vue'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { fetchInsight } from '../lib/api'
import { useForumStore } from '../stores/forum'

const route = useRoute()
const router = useRouter()
const forumStore = useForumStore()
const insightId = computed(() => Number(route.params.id))
const insightQuery = useQuery({
  queryKey: computed(() => ['insight-detail', insightId.value]),
  queryFn: () => fetchInsight(insightId.value),
})
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <article v-if="insightQuery.data.value" class="insight-article">
      <div class="breadcrumb">首页 / 数据建议 / 选科组合趋势</div>
      <h1>{{ insightQuery.data.value.combination }}</h1>
      <p class="article-lead">{{ insightQuery.data.value.advice }}</p>
      <div class="insight-score insight-score-wide">
        <span><TrendingUp :size="18" /><strong>{{ insightQuery.data.value.heat }}</strong> 热度</span>
        <span><Gauge :size="18" /><strong>{{ insightQuery.data.value.matchRate }}%</strong> 匹配度</span>
        <span><BarChart3 :size="18" /><strong>{{ insightQuery.data.value.trend }}</strong> 趋势</span>
      </div>
      <p class="article-body">{{ insightQuery.data.value.details }}</p>
      <div class="publish-strip">
        <span>想结合你的成绩和目标专业继续讨论？</span>
        <button class="primary-wide" @click="forumStore.openPublish('question')">发布提问</button>
      </div>
    </article>
  </main>
</template>
