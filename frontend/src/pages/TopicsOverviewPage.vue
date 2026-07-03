<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { ChevronLeft, Eye, Hash, MessageSquare, PenLine, Search, TrendingUp } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { apiDataEnabled, fetchTopics } from '../lib/api'
import { useForumStore } from '../stores/forum'

const router = useRouter()
const forumStore = useForumStore()
const topicsQuery = useQuery({
  queryKey: ['topics-overview'],
  queryFn: fetchTopics,
  enabled: apiDataEnabled,
})
const topicCards = computed(() =>
  (topicsQuery.data.value ?? []).map((topic, index) => ({
    ...topic,
    tone: index % 4,
    prompts: [
      `${topic.title} 有哪些真实坑？`,
      `同省同组合怎么判断？`,
      `老师/学长怎么看 ${topic.title}？`,
    ],
  })),
)

function searchTopic(slug: string) {
  router.push(`/topics/${slug}`)
}
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="overview-hero">
      <div>
        <div class="breadcrumb">首页 / 热门话题</div>
        <h1>热门话题广场</h1>
        <p>按浏览量、讨论量和主题方向聚合选科问题，适合快速进入同类学生和家长正在讨论的场景。</p>
        <div class="overview-metrics">
          <span><TrendingUp :size="18" /> 热门聚合</span>
          <span><MessageSquare :size="18" /> 可追问</span>
          <span><Eye :size="18" /> 长尾搜索</span>
        </div>
      </div>
      <button class="primary-wide compact" type="button" @click="forumStore.openPublish('question')">
        <PenLine :size="16" /> 发起话题
      </button>
    </section>

    <section class="topic-card-grid">
      <article
        v-for="topic in topicCards"
        :key="topic.slug"
        class="topic-discovery-card"
        :class="`tone-${topic.tone}`"
      >
        <RouterLink :to="`/topics/${topic.slug}`">
          <span class="topic-hash"><Hash :size="18" /></span>
          <small>{{ (topic.viewsCount / 1000).toFixed(1) }}k 浏览 · {{ topic.postsCount }} 篇讨论</small>
          <h2>{{ topic.title }}</h2>
          <p>{{ topic.summary }}</p>
        </RouterLink>
        <div class="topic-prompt-list">
          <button v-for="prompt in topic.prompts" :key="prompt" type="button" @click="searchTopic(topic.slug)">
            <Search :size="14" /> {{ prompt }}
          </button>
        </div>
      </article>
    </section>
  </main>
</template>
