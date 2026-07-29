<script setup lang="ts">
import { BarChart3, ChevronRight, ClipboardCheck, Sparkles, TrendingUp } from '@lucide/vue'
import { computed } from 'vue'
import type { SubjectInsight, Topic } from '../types/forum'

const props = defineProps<{
  insights: SubjectInsight[]
  topics: Topic[]
}>()

const palette = ['#10b981', '#2563eb', '#38bdf8', '#f59e0b', '#ef4444', '#8b5cf6', '#64748b']
const trendItems = computed(() => props.insights)
const trendTotal = computed(() => trendItems.value.reduce((sum, insight) => sum + insight.heat, 0) || 1)

const donutStyle = computed(() => {
  let cursor = 0
  const slices = trendItems.value.map((insight, index) => {
    const start = cursor
    const span = (insight.heat / trendTotal.value) * 100
    cursor += span
    return `${palette[index % palette.length]} ${start}% ${Math.min(cursor, 100)}%`
  })

  return {
    background: `conic-gradient(${slices.join(', ')})`,
  }
})

</script>

<template>
  <aside class="insight-panel">
    <section class="insight-box insight-trend-section">
      <div class="panel-title-row">
        <h2>广东招生计划选科观察</h2>
        <span class="insight-section-badge"><TrendingUp :size="14" /> 官方数据</span>
      </div>
      <p class="caption">2025专科第一次征集志愿 · 计划要求分布</p>
      <div class="trend-chart" aria-label="招生计划选科要求分布图">
        <div class="trend-donut" :style="donutStyle"></div>
        <div class="trend-legend">
          <RouterLink v-for="(insight, index) in trendItems" :key="insight.id" :to="`/insights/${insight.id}`">
            <i :style="{ background: palette[index % palette.length] }"></i>
            <span>{{ insight.combination }}</span>
            <strong>{{ insight.heat }} 个</strong>
          </RouterLink>
        </div>
      </div>
      <a v-if="trendItems[0]" class="insight-more" :href="trendItems[0].sourceUrl" target="_blank" rel="noreferrer">
        来源：{{ trendItems[0].sourceName }} <ChevronRight :size="14" />
      </a>
      <RouterLink class="insight-more" to="/insights">查看全部趋势 <ChevronRight :size="14" /></RouterLink>
    </section>

    <section class="insight-box insight-topics-section">
      <div class="panel-title-row">
        <h2>热门话题</h2>
        <span class="insight-section-index">02</span>
      </div>
      <p class="caption">广东学生和家长正在讨论</p>
      <div class="topic-list">
        <RouterLink v-for="topic in topics" :key="topic.slug" :to="`/topics/${topic.slug}`">
          <span># {{ topic.title }}</span>
          <strong>{{ (topic.viewsCount / 1000).toFixed(1) }}k浏览</strong>
        </RouterLink>
      </div>
      <RouterLink class="insight-more" to="/topics">查看全部话题 <ChevronRight :size="14" /></RouterLink>
    </section>

    <section class="insight-box insight-advice-section">
      <div class="panel-title-row">
        <h2>选科建议</h2>
        <span class="insight-section-index">03</span>
      </div>
      <p class="caption">结合趋势快速排查组合风险</p>
      <div class="advice-stack">
        <RouterLink v-for="(insight, index) in insights.slice(0, 3)" :key="insight.id" class="advice-row" :to="`/insights/${insight.id}`">
          <span class="advice-icon" :class="`tone-${index}`">
            <Sparkles v-if="index === 0" :size="18" />
            <ClipboardCheck v-else-if="index === 1" :size="18" />
            <BarChart3 v-else :size="18" />
          </span>
          <span>
            <strong>{{ insight.combination }} · {{ insight.trend }}</strong>
            <small>{{ insight.advice }}</small>
          </span>
          <TrendingUp :size="16" />
        </RouterLink>
      </div>
      <RouterLink class="insight-more" to="/advice">查看全部建议 <ChevronRight :size="14" /></RouterLink>
    </section>
  </aside>
</template>
