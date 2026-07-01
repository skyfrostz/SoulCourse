<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { ChevronLeft, ClipboardCheck, Lightbulb, PenLine, Sparkles } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { fetchInsights } from '../lib/api'
import { sampleInsights } from '../lib/sampleData'
import { useForumStore } from '../stores/forum'

const router = useRouter()
const forumStore = useForumStore()
const insightsQuery = useQuery({
  queryKey: ['advice-overview'],
  queryFn: fetchInsights,
})
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="overview-hero advice-hero">
      <div class="breadcrumb">首页 / 精选建议</div>
      <h1>精选建议库</h1>
      <p>把组合适配、学习压力、专业覆盖和填报风险拆成可执行建议，方便学生和家长逐项核对。</p>
      <button class="primary-wide compact" @click="forumStore.openPublish('question')">
        <PenLine :size="16" /> 发布我的选科问题
      </button>
    </section>

    <section class="advice-board">
      <RouterLink
        v-for="(insight, index) in insightsQuery.data.value ?? sampleInsights"
        :key="insight.id"
        class="advice-detail-card"
        :to="`/insights/${insight.id}`"
      >
        <span class="advice-icon" :class="`tone-${index % 3}`">
          <Sparkles v-if="index % 3 === 0" :size="20" />
          <ClipboardCheck v-else-if="index % 3 === 1" :size="20" />
          <Lightbulb v-else :size="20" />
        </span>
        <span>
          <small>{{ insight.trend }}</small>
          <strong>{{ insight.combination }}</strong>
          <p>{{ insight.details }}</p>
        </span>
      </RouterLink>
    </section>
  </main>
</template>
