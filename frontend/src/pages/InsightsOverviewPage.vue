<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, ref, watch } from 'vue'
import { BarChart3, ChevronLeft, Search, ShieldCheck, TrendingUp } from '@lucide/vue'
import { useRoute, useRouter } from 'vue-router'
import { apiDataEnabled, fetchInsights, fetchProvinceCoverage, fetchPublishedPolicies } from '../lib/api'
import { useGlobalSearch } from '../composables/useGlobalSearch'
import { useOnlineState } from '../composables/useOnlineState'
import type { SubjectInsight } from '../types/forum'

const router = useRouter()
const route = useRoute()
const { runSearch } = useGlobalSearch()
const { isOffline } = useOnlineState()
const insightsQuery = useQuery({
  queryKey: ['insights-overview'],
  queryFn: fetchInsights,
  enabled: apiDataEnabled,
})
const provincesQuery = useQuery({
  queryKey: ['real-data', 'provinces'],
  queryFn: fetchProvinceCoverage,
  enabled: apiDataEnabled,
})
const policiesQuery = useQuery({
  queryKey: ['real-data', 'policies'],
  queryFn: fetchPublishedPolicies,
  enabled: apiDataEnabled,
})
const hasDataError = computed(() => insightsQuery.isError.value || provincesQuery.isError.value || policiesQuery.isError.value)
const isDataLoading = computed(() => insightsQuery.isLoading.value || provincesQuery.isLoading.value || policiesQuery.isLoading.value)
const mode = ref<'count' | 'share'>(route.query.mode === 'share' ? 'share' : 'count')
const modes = [
  { value: 'count', label: '计划数排序', shortLabel: '计划数', icon: TrendingUp },
  { value: 'share', label: '数据集占比', shortLabel: '占比', icon: BarChart3 },
] as const

function scoreFor(insight: SubjectInsight) {
  if (mode.value === 'share') return insight.matchRate
  const maximum = Math.max(...(insightsQuery.data.value ?? []).map((item) => item.heat), 1)
  return insight.heat / maximum * 100
}

const insightCards = computed(() => {
  const source = [...(insightsQuery.data.value ?? [])]
  if (mode.value === 'share') return source.sort((a, b) => b.matchRate - a.matchRate)
  return source.sort((a, b) => b.heat - a.heat)
})

watch(
  () => route.query.mode,
  (value) => {
    mode.value = value === 'share' ? 'share' : 'count'
  },
)

function setMode(value: 'count' | 'share') {
  mode.value = value
  router.replace({ path: '/insights', query: value === 'count' ? {} : { mode: value } })
}

function searchCombination(combination: string) {
  void runSearch(combination)
}

function refetchData() {
  void insightsQuery.refetch()
  void provincesQuery.refetch()
  void policiesQuery.refetch()
}
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="overview-hero insights-overview-hero">
      <div class="insights-hero-copy">
        <div class="breadcrumb">首页 / 选科组合趋势</div>
        <h1>官方选科要求数据中心</h1>
        <p>只展示能够从考试招生机构原始材料复算的指标，并把统计范围、来源和抓取时间一并公开。</p>
      </div>
      <div class="insights-sort-panel" aria-label="趋势排序">
        <span>排序方式</span>
        <div class="insights-sort-control">
          <button
            v-for="item in modes"
            :key="item.value"
            type="button"
            :class="{ active: mode === item.value }"
            @click="setMode(item.value)"
          >
            <component :is="item.icon" :size="17" />
            {{ item.label }}
          </button>
        </div>
        <div class="insights-sort-summary">
          当前按 <strong>{{ modes.find((item) => item.value === mode)?.shortLabel }}</strong> 展示
        </div>
      </div>
    </section>

    <section v-if="hasDataError || isOffline" class="empty-state public-page-state">
      <h2>{{ isOffline ? '当前网络不可用' : '趋势数据暂时无法加载' }}</h2>
      <p>{{ isOffline ? '恢复网络后再重试，页面不会生成模拟趋势。' : '趋势、覆盖范围或政策来源没有同步成功，请稍后重试。' }}</p>
      <button class="primary-wide compact" type="button" @click="refetchData">重试</button>
    </section>

    <section v-else-if="isDataLoading" class="empty-state public-page-state" aria-live="polite">
      <p>正在加载已复核趋势数据...</p>
    </section>

    <template v-else>
    <section class="insights-chart-panel" aria-labelledby="insights-chart-title">
      <div class="insights-chart-heading">
        <div>
          <span>趋势图表</span>
          <h2 id="insights-chart-title">选科组合{{ mode === 'count' ? '计划数' : '数据集占比' }}排行</h2>
          <p>基于当前已复核的官方来源数据，点击组合可查看指标明细与来源。</p>
        </div>
        <BarChart3 :size="24" aria-hidden="true" />
      </div>
      <div class="insights-bar-chart" role="list" aria-label="选科组合排行图">
        <RouterLink
          v-for="insight in insightCards.slice(0, 8)"
          :key="`chart-${insight.id}`"
          :to="`/insights/${insight.id}`"
          class="insight-bar-row"
          role="listitem"
        >
          <span class="insight-bar-label">{{ insight.combination }}</span>
          <span class="insight-bar-track">
            <span class="insight-bar-fill" :style="{ width: `${Math.min(scoreFor(insight), 100)}%` }"></span>
          </span>
          <strong>{{ mode === 'count' ? `${insight.heat} ${insight.unit}` : `${insight.matchRate}%` }}</strong>
        </RouterLink>
      </div>
    </section>
    <section id="trend-board" class="insight-feature-grid xhs-trend-grid">
      <article v-for="insight in insightCards" :key="insight.id" class="insight-feature-card">
        <RouterLink :to="`/insights/${insight.id}`">
          <small>{{ insight.trend }}</small>
          <h2>{{ insight.combination }}</h2>
          <p>{{ insight.advice }}</p>
          <div class="feature-meter">
            <span :style="{ width: `${Math.min(scoreFor(insight), 100)}%` }"></span>
          </div>
          <div class="overview-score">
            <span>{{ insight.unit }} {{ insight.heat }}</span>
            <span>占数据集 {{ insight.matchRate }}%</span>
            <span>{{ insight.dataYear }} · {{ insight.province }}</span>
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
        <h2>省级官方来源与公开情况</h2>
        <p>仅展示省份覆盖接口返回的复核状态与方法说明；没有可靠组合级数据时明确标记缺失。</p>
      </div>
      <div class="data-lab-grid">
        <article v-for="item in provincesQuery.data.value ?? []" :key="item.province" class="data-chart-card">
          <div>
            <small>{{ item.province }} · {{ item.coverageStatus === 'verified' ? '已复核数据' : '暂无已复核数据' }}</small>
            <h3>{{ item.methodology }}</h3>
          </div>
          <div class="empty-state compact-empty">
            <strong>{{ item.coverageStatus === 'verified' ? '已纳入复核范围' : '暂无官方公开数据' }}</strong>
            <p>{{ item.coverageStatus === 'verified' ? '具体组合指标以同省份、同年份的已发布洞察为准。' : '不使用专业覆盖率、门户分类或公式生成值代替考生组合热度。' }}</p>
          </div>
          <p class="source-note"><ShieldCheck :size="15" /> {{ item.capturedAt ? `采集于 ${new Date(item.capturedAt).toLocaleDateString('zh-CN')}` : '请从官方入口核对最新公告' }}</p>
        </article>
        <article v-if="!provincesQuery.isLoading.value && !(provincesQuery.data.value?.length)" class="empty-state compact-empty">
          <strong>暂无已复核数据</strong>
          <p>省份覆盖接口暂未返回可公开的数据记录。</p>
        </article>
      </div>
    </section>

    <section class="takeaway-panel">
      <article v-for="policy in policiesQuery.data.value ?? []" :key="policy.id">
        <strong>{{ policy.title }}</strong>
        <p>{{ policy.summary || policy.methodology }}</p>
        <a :href="policy.source.url || policy.url" target="_blank" rel="noreferrer">{{ policy.source.name }}</a>
      </article>
      <article v-if="!policiesQuery.isLoading.value && !(policiesQuery.data.value?.length)">
        <strong>暂无已复核政策结论</strong>
        <p>政策内容将在完成来源核验后公开展示。</p>
      </article>
    </section>
    </template>
  </main>
</template>
