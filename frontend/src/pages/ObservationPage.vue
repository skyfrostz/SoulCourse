<script setup lang="ts">
import InsightPanel from '../components/InsightPanel.vue'
import { useForumData } from '../composables/useForumData'
import { useOnlineState } from '../composables/useOnlineState'

const { insights, topics, observationLoading, observationHasError, refetchObservations } = useForumData()
const { isOffline } = useOnlineState()
</script>

<template>
  <main class="mobile-observation-page">
    <section v-if="observationHasError || isOffline" class="empty-state detail-empty-state">
      <h1>{{ isOffline ? '当前网络不可用' : '观察数据暂时无法加载' }}</h1>
      <p>{{ isOffline ? '恢复网络后再重试，观察站不会生成模拟结论。' : '趋势或话题数据没有同步成功，请检查网络后重试。' }}</p>
      <button class="primary-wide compact" type="button" @click="refetchObservations">重新加载</button>
    </section>
    <section v-else-if="observationLoading" class="empty-state detail-empty-state" aria-live="polite">
      <p>正在加载广东选科观察...</p>
    </section>
    <section v-else-if="!insights.length && !topics.length" class="empty-state detail-empty-state">
      <h1>暂时没有已发布观察数据</h1>
      <p>这里只展示真实发布的趋势和话题，不生成模拟结论。</p>
    </section>
    <InsightPanel v-else :insights="insights" :topics="topics" />
  </main>
</template>
