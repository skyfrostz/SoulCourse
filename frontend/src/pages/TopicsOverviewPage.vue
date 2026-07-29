<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { ChevronLeft, Eye, Hash, MessageSquare, PenLine, Search, TrendingUp } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { apiDataEnabled, fetchTopic, fetchTopics } from '../lib/api'
import { useForumStore } from '../stores/forum'

const router = useRouter()
const forumStore = useForumStore()
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

const fallbackQuestions: Record<string, string[]> = {
  物理方向: ['物理成绩波动大，还适合选物理方向吗？', '目标专业没定，物理方向该怎么留余地？', '物理方向不同组合的学习强度差多少？'],
  历史方向: ['历史方向填志愿时最容易忽略什么限制？', '偏文科但数学不错，组合该怎么取舍？', '历史方向如何提前核对目标专业要求？'],
  化学重要性: ['哪些专业明确要求再选化学？', '化学成绩一般，保专业覆盖还是保赋分？', '不同省份对化学的选科要求差别大吗？'],
  选科时间线: ['高一什么时候开始记录单科排名最有用？', '正式选科前要做哪几轮组合验证？', '选科确认后还能调整吗，成本有多大？'],
  提分方法: ['选科后应该优先补短板还是放大优势？', '赋分科目怎样判断真实提分空间？', '不同组合如何安排一周复习时间？'],
  家长选科: ['家长和孩子意见相反时怎么做验证？', '家长该看分数、排名还是专业方向？', '咨询老师前应该准备哪些成绩信息？'],
}

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
    for (const label of fallbackQuestions[topic.topicTag] ?? []) {
      if (prompts.length === 3) break
      if (usedPrompts.has(label)) continue
      usedPrompts.add(label)
      prompts.push({ label })
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
          <button v-for="prompt in topic.prompts" :key="prompt.label" type="button" @click="openPrompt(topic.slug, prompt.postId)">
            <Search :size="14" /> {{ prompt.label }}
          </button>
        </div>
      </article>
    </section>
  </main>
</template>
