<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { BarChart3, ChevronLeft, Gauge, TrendingUp } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { fetchInsights } from '../lib/api'
import { policyTakeaways, requirementData } from '../lib/realData'
import { sampleInsights } from '../lib/sampleData'

const router = useRouter()
const insightsQuery = useQuery({
  queryKey: ['insights-overview'],
  queryFn: fetchInsights,
})

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
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="overview-hero">
      <div class="breadcrumb">首页 / 选科组合趋势</div>
      <h1>选科组合趋势中心</h1>
      <p>把组合热度、专业覆盖、学习强度和后续讨论放在同一个页面里比较，避免只凭“热门”做决定。</p>
      <div class="overview-metrics">
        <span><TrendingUp :size="18" /> 热度排序</span>
        <span><Gauge :size="18" /> 匹配度</span>
        <span><BarChart3 :size="18" /> 专业覆盖</span>
      </div>
    </section>

    <section class="data-lab">
      <div class="section-heading">
        <span>真实数据看板</span>
        <h2>不同省份选考要求对比</h2>
        <p>以下数据来自公开考试院/阳光高考信息及基于考试院目录的公开统计，适合用来判断“物化”要求的真实权重。</p>
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

    <section class="overview-grid">
      <RouterLink
        v-for="insight in insightsQuery.data.value ?? sampleInsights"
        :key="insight.id"
        class="overview-card"
        :to="`/insights/${insight.id}`"
      >
        <small>{{ insight.trend }}</small>
        <h2>{{ insight.combination }}</h2>
        <p>{{ insight.advice }}</p>
        <div class="overview-score">
          <span>热度 {{ insight.heat }}</span>
          <span>匹配 {{ insight.matchRate }}%</span>
        </div>
      </RouterLink>
    </section>
  </main>
</template>
