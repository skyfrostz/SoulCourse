<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, ref, watch } from 'vue'
import { BarChart3, ChevronLeft, Search, ShieldCheck, TrendingUp } from '@lucide/vue'
import { useRoute, useRouter } from 'vue-router'
import { apiDataEnabled, fetchInsights } from '../lib/api'
import { policyTakeaways, requirementData } from '../lib/realData'
import { useForumStore } from '../stores/forum'
import type { SubjectInsight } from '../types/forum'

const router = useRouter()
const route = useRoute()
const forumStore = useForumStore()
const insightsQuery = useQuery({
  queryKey: ['insights-overview'],
  queryFn: fetchInsights,
  enabled: apiDataEnabled,
})
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

function donutStyle(index: number) {
  const data = requirementData[index]
  if (!data?.slices.length) return { background: '#e2e8f0' }
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
        <p>广东展示已复算的官方招生计划数据；其他省份没有可靠组合级数据时明确标记缺失，只保留官方查询入口。</p>
      </div>
      <div class="data-lab-grid">
        <article v-for="(item, index) in requirementData" :key="item.province" class="data-chart-card">
          <div>
            <small>{{ item.province }} · {{ item.total ? `${item.total} 个计划数` : '暂无组合级统计' }}</small>
            <h3>{{ item.note }}</h3>
          </div>
          <div v-if="item.slices.length" class="source-donut-row">
            <span class="source-donut" :style="donutStyle(index)"></span>
            <div class="source-legend">
              <span v-for="slice in item.slices" :key="slice.label">
                <i :style="{ background: slice.color }"></i>
                {{ slice.label }}
                <strong>{{ slice.value }}%</strong>
              </span>
            </div>
          </div>
          <div v-else class="empty-state compact-empty">
            <strong>暂无官方公开数据</strong>
            <p>不使用专业覆盖率、门户分类或公式生成值代替考生组合热度。</p>
          </div>
          <a :href="item.source.url" target="_blank" rel="noreferrer">来源：{{ item.source.publisher }}</a>
          <p class="source-note"><ShieldCheck :size="15" /> {{ item.capturedAt ? `抓取于 ${item.capturedAt}` : '请从官方入口核对最新公告' }}</p>
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
