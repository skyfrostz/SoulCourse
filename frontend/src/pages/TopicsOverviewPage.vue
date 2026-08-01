<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { ChevronLeft, Eye, Hash, MessageSquare, PenLine, Search, TrendingUp } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { apiDataEnabled, fetchTopic, fetchTopics } from '../lib/api'
import { useOnlineState } from '../composables/useOnlineState'
import { useForumStore } from '../stores/forum'

const router = useRouter()
const forumStore = useForumStore()
const { isOffline } = useOnlineState()
const topicsQuery = useQuery({
  queryKey: ['topics-overview', 'real-prompts'],
  queryFn: async () => {
    const topics = await fetchTopics()
    return Promise.all(topics.map(async (topic) => {
      try {
        const detail = await fetchTopic(topic.slug)
        return { topic, posts: detail.posts }
      } catch {
        return { topic, posts: [] }
      }
    }))
  },
  enabled: apiDataEnabled,
})

const topicCards = computed(() => {
  const usedPrompts = new Set<string>()
  return (topicsQuery.data.value ?? []).map(({ topic, posts }, index) => {
    const prompts: Array<{ label: string; postId?: number }> = []
    for (const post of posts) {
      const label = post.title.trim()
      if (!label || usedPrompts.has(label)) continue
      usedPrompts.add(label)
      prompts.push({ label, postId: post.id })
      if (prompts.length === 3) break
    }
    return { ...topic, tone: index % 4, prompts }
  })
})

function openPrompt(slug: string, postId?: number) {
  router.push(postId ? `/posts/${postId}` : `/topics/${slug}`)
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

    <section v-if="topicsQuery.isError.value || isOffline" class="empty-state detail-empty-state">
      <h2>{{ isOffline ? '当前网络不可用' : '话题暂时无法加载' }}</h2>
      <p>{{ isOffline ? '恢复网络后再刷新，本站不会用模拟话题替代真实讨论。' : '请检查网络后重试，本站不会用模拟话题替代真实讨论。' }}</p>
      <button class="primary-wide compact" type="button" @click="topicsQuery.refetch()">重新加载</button>
    </section>

    <section v-else-if="topicsQuery.isLoading.value" class="topic-card-grid" aria-label="正在加载话题">
      <article v-for="index in 3" :key="index" class="topic-discovery-card topic-loading-card" aria-hidden="true">
        <span class="topic-skeleton-line" />
        <span class="topic-skeleton-line short" />
        <span class="topic-skeleton-block" />
      </article>
    </section>

    <section v-else-if="!topicCards.length" class="empty-state detail-empty-state">
      <Hash :size="30" />
      <h2>暂时没有已发布话题</h2>
      <p>话题会在真实讨论达到发布条件后出现，你可以先发起一个选科问题。</p>
      <button class="primary-wide compact" type="button" @click="forumStore.openPublish('question')">发起话题</button>
    </section>

    <section v-else class="topic-card-grid">
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
          <button v-for="prompt in topic.prompts" :key="prompt.label" type="button" @click="openPrompt(topic.slug, prompt.postId)">
            <Search :size="14" /> {{ prompt.label }}
          </button>
        </div>
      </article>
    </section>
  </main>
</template>
