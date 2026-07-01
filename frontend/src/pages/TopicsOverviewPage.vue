<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { ChevronLeft, Eye, MessageSquare, Hash } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { fetchTopics } from '../lib/api'

const router = useRouter()
const topicsQuery = useQuery({
  queryKey: ['topics-overview'],
  queryFn: fetchTopics,
})
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="overview-hero">
      <div class="breadcrumb">首页 / 热门话题</div>
      <h1>热门话题广场</h1>
      <p>按浏览量、讨论量和主题方向聚合选科问题，适合快速进入同类学生和家长正在讨论的场景。</p>
    </section>

    <section class="topic-directory">
      <RouterLink
        v-for="topic in topicsQuery.data.value ?? []"
        :key="topic.slug"
        class="topic-directory-row"
        :to="`/topics/${topic.slug}`"
      >
        <span class="topic-hash"><Hash :size="18" /></span>
        <span>
          <strong>{{ topic.title }}</strong>
          <small>{{ topic.summary }}</small>
        </span>
        <span><Eye :size="16" /> {{ topic.viewsCount }}</span>
        <span><MessageSquare :size="16" /> {{ topic.postsCount }}</span>
      </RouterLink>
    </section>
  </main>
</template>
