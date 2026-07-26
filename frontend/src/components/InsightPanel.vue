<script setup lang="ts">
import { BarChart3, ChevronRight, ClipboardCheck, Sparkles, TrendingUp } from '@lucide/vue'
import { computed, ref } from 'vue'
import type { SubjectInsight, Topic } from '../types/forum'

const props = defineProps<{
  insights: SubjectInsight[]
  topics: Topic[]
}>()

const palette = ['#10b981', '#2563eb', '#38bdf8', '#f59e0b', '#ef4444']
const activeTab = ref<'trend' | 'topics' | 'advice'>('trend')
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
        <h2>广东选科观察</h2>
      </div>
      <div class="insight-tabs" role="tablist" aria-label="观察内容">
        <button type="button" :aria-pressed="activeTab === 'trend'" @click="activeTab = 'trend'">趋势</button>
        <button type="button" :aria-pressed="activeTab === 'topics'" @click="activeTab = 'topics'">话题</button>
        <button type="button" :aria-pressed="activeTab === 'advice'" @click="activeTab = 'advice'">建议</button>
      </div>

      <template v-if="activeTab === 'trend'">
        <p class="caption">2026届考生组合热度</p>
        <div class="trend-chart" aria-label="选科组合趋势图">
          <div class="trend-donut" :style="donutStyle"></div>
          <div class="trend-legend">
            <RouterLink v-for="(insight, index) in trendItems" :key="insight.id" :to="`/insights/${insight.id}`">
              <i :style="{ background: palette[index % palette.length] }"></i>
              <span>{{ insight.combination }}</span>
              <strong>{{ insight.heat }}</strong>
            </RouterLink>
          </div>
        </div>
        <RouterLink class="insight-more" to="/insights">查看全部趋势 <ChevronRight :size="14" /></RouterLink>
      </template>

      <template v-else-if="activeTab === 'topics'">
        <div class="topic-list">
          <RouterLink v-for="topic in topics" :key="topic.slug" :to="`/topics/${topic.slug}`">
            <span># {{ topic.title }}</span>
            <strong>{{ (topic.viewsCount / 1000).toFixed(1) }}k浏览</strong>
          </RouterLink>
        </div>
        <RouterLink class="insight-more" to="/topics">查看全部话题 <ChevronRight :size="14" /></RouterLink>
      </template>

      <template v-else>
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
      </template>
    </section>
  </aside>
</template>
