<script setup lang="ts">
import { BarChart3, ChevronRight, ClipboardCheck, Sparkles, TrendingUp } from '@lucide/vue'
import { computed } from 'vue'
import type { SubjectInsight, Topic } from '../types/forum'

const props = defineProps<{
  insights: SubjectInsight[]
  topics: Topic[]
}>()

const palette = ['#10b981', '#2563eb', '#38bdf8', '#f59e0b', '#ef4444']
const trendItems = computed(() => props.insights.slice(0, 5))
const trendTotal = computed(() => trendItems.value.reduce((sum, insight) => sum + insight.heat, 0) || 1)

const donutStyle = computed(() => {
  let cursor = 0
  const slices = trendItems.value.map((insight, index) => {
    const start = cursor
    const span = Math.max((insight.heat / trendTotal.value) * 100, 6)
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
    <section class="insight-box">
      <div class="panel-title-row">
        <h2>选科组合趋势</h2>
        <RouterLink class="ghost-link" to="/insights">更多 <ChevronRight :size="14" /></RouterLink>
      </div>
      <p class="caption">2026届考生选择趋势（真实接口）</p>
      <div class="trend-chart" aria-label="选科组合趋势图">
        <div class="trend-donut" :style="donutStyle"></div>
        <div class="trend-legend">
          <RouterLink
            v-for="(insight, index) in trendItems"
            :key="insight.id"
            :to="`/insights/${insight.id}`"
          >
            <i :style="{ background: palette[index % palette.length] }"></i>
            <span>{{ insight.combination }}</span>
            <strong>{{ insight.heat }}</strong>
          </RouterLink>
        </div>
      </div>
      <p class="data-source">点击下方精选建议可查看分析详情</p>
    </section>

    <section class="insight-box">
      <div class="panel-title-row">
        <h2>热门话题</h2>
        <RouterLink class="ghost-link" to="/topics">更多 <ChevronRight :size="14" /></RouterLink>
      </div>
      <div class="topic-list">
        <RouterLink v-for="topic in topics" :key="topic.slug" :to="`/topics/${topic.slug}`">
          <span># {{ topic.title }}</span>
          <strong>{{ (topic.viewsCount / 1000).toFixed(1) }}k浏览</strong>
        </RouterLink>
      </div>
    </section>

    <section class="insight-box">
      <div class="panel-title-row">
        <h2>精选建议</h2>
        <RouterLink class="ghost-link" to="/advice">更多 <ChevronRight :size="14" /></RouterLink>
      </div>
      <div class="advice-stack">
        <RouterLink
          v-for="(insight, index) in insights.slice(0, 3)"
          :key="insight.id"
          class="advice-row"
          :to="`/insights/${insight.id}`"
        >
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
    </section>
  </aside>
</template>
