<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, ref, watch } from 'vue'
import { BarChart3, ChevronLeft, Gauge, Search, ShieldCheck, TrendingUp } from '@lucide/vue'
import { useRoute, useRouter } from 'vue-router'
import { apiDataEnabled, fetchInsights } from '../lib/api'
import { policyTakeaways, requirementData } from '../lib/realData'
import { sampleInsights } from '../lib/sampleData'
import { useForumStore } from '../stores/forum'

const router = useRouter()
const route = useRoute()
const forumStore = useForumStore()
const insightsQuery = useQuery({
  queryKey: ['insights-overview'],
  queryFn: fetchInsights,
  enabled: apiDataEnabled,
})
const mode = ref<'heat' | 'match' | 'coverage'>(
  route.query.mode === 'match' || route.query.mode === 'coverage' ? route.query.mode : 'heat',
)
const modes = [
  { value: 'heat', label: '热度' },
  { value: 'match', label: '匹配度' },
  { value: 'coverage', label: '专业覆盖' },
] as const
const insightCards = computed(() => {
  const apiInsights = insightsQuery.data.value ?? []
  const source = [...apiInsights, ...sampleInsights.filter((item) => !apiInsights.some((apiItem) => apiItem.id === item.id))]
  if (mode.value === 'match') return source.sort((a, b) => b.matchRate - a.matchRate)
  if (mode.value === 'coverage') return source.sort((a, b) => b.matchRate + b.heat * 0.2 - (a.matchRate + a.heat * 0.2))
  return source.sort((a, b) => b.heat - a.heat)
})

watch(
  () => route.query.mode,
  (value) => {
    if (value === 'match' || value === 'coverage' || value === 'heat') mode.value = value
  },
)

function donutStyle(index: number) {
  const data = requirementData[index]
  let cursor = 0
  const stops = data.slices.map((slice) => {
    const start = cursor
    cursor += slice.value
    return `${slice.color} ${start}% ${Math.min(cursor, 100)}%`
  })
  return { background: `conic-gradient(${stops.join(', ')})` }
}

function searchCombination(combination: string) {
  forumStore.setKeyword(combination)
  router.push('/')
}
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="overview-hero">
      <div class="breadcrumb">首页 / 选科组合趋势</div>
      <h1>选科组合趋势中心</h1>
      <p>把组合热度、专业覆盖、学习强度和后续讨论放在同一个页面里比较，避免只凭“热门”做决定。</p>
      <div class="overview-metrics">
        <RouterLink :to="{ path: '/insights', query: { mode: 'heat' } }"><TrendingUp :size="18" /> 热度排序</RouterLink>
        <RouterLink :to="{ path: '/insights', query: { mode: 'match' } }"><Gauge :size="18" /> 匹配度</RouterLink>
        <RouterLink :to="{ path: '/insights', query: { mode: 'coverage' } }"><BarChart3 :size="18" /> 专业覆盖</RouterLink>
      </div>
    </section>

    <nav class="content-lens-tabs" aria-label="趋势排序">
      <button
        v-for="item in modes"
        :key="item.value"
        type="button"
        :class="{ active: mode === item.value }"
        @click="mode = item.value"
      >
        {{ item.label }}
      </button>
    </nav>

    <section id="trend-board" class="insight-feature-grid xhs-trend-grid">
      <article v-for="insight in insightCards" :key="insight.id" class="insight-feature-card">
        <RouterLink :to="`/insights/${insight.id}`">
          <small>{{ insight.trend }}</small>
          <h2>{{ insight.combination }}</h2>
          <p>{{ insight.advice }}</p>
          <div class="feature-meter">
            <span :style="{ width: `${Math.min(insight.matchRate, 100)}%` }"></span>
          </div>
          <div class="overview-score">
            <span>热度 {{ insight.heat }}</span>
            <span>匹配 {{ insight.matchRate }}%</span>
          </div>
        </RouterLink>
        <button type="button" @click="searchCombination(insight.combination)">
          <Search :size="15" /> 搜同款经验
        </button>
      </article>
    </section>

    <section class="data-lab">
      <div class="section-heading">
        <span>真实数据看板</span>
        <h2>34 个省级招生政策与选考入口</h2>
        <p>前三张为公开统计数据，其余为省级考试院/港澳台招生入口核对卡。适合先建立全国视野，再回到本省最新目录和高校章程逐条核对。</p>
      </div>
      <div class="data-lab-grid">
        <article v-for="(item, index) in requirementData" :key="item.province" class="data-chart-card">
          <div>
            <small>{{ item.province }} · {{ item.total ? `${item.total} 条专业数据` : '公开目录' }}</small>
            <h3>{{ item.note }}</h3>
          </div>
          <div class="source-donut-row">
            <span class="source-donut" :style="donutStyle(index)"></span>
            <div class="source-legend">
              <span v-for="slice in item.slices" :key="slice.label">
                <i :style="{ background: slice.color }"></i>
                {{ slice.label }}
                <strong>{{ slice.value }}%</strong>
              </span>
            </div>
          </div>
          <a :href="item.source.url" target="_blank" rel="noreferrer">来源：{{ item.source.publisher }}</a>
          <p class="source-note"><ShieldCheck :size="15" /> 建议与本省最新考试院目录交叉核对。</p>
        </article>
      </div>
    </section>

    <section class="takeaway-panel">
      <article v-for="takeaway in policyTakeaways" :key="takeaway.title">
        <strong>{{ takeaway.title }}</strong>
        <p>{{ takeaway.body }}</p>
        <a :href="takeaway.source.url" target="_blank" rel="noreferrer">{{ takeaway.source.publisher }}</a>
      </article>
    </section>
  </main>
</template>
