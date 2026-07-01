<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { ChevronLeft, Eye, MessageSquare } from '@lucide/vue'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PostCard from '../components/PostCard.vue'
import { fetchTopic } from '../lib/api'

const route = useRoute()
const router = useRouter()
const slug = computed(() => String(route.params.slug))
const topicQuery = useQuery({
  queryKey: computed(() => ['topic-detail', slug.value]),
  queryFn: () => fetchTopic(slug.value),
})
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section v-if="topicQuery.data.value" class="topic-hero">
      <div class="breadcrumb">首页 / 热门话题</div>
      <h1># {{ topicQuery.data.value.topic.title }}</h1>
      <p>{{ topicQuery.data.value.topic.summary }}</p>
      <div class="article-actions">
        <span><Eye :size="17" /> {{ topicQuery.data.value.topic.viewsCount }} 浏览</span>
        <span><MessageSquare :size="17" /> {{ topicQuery.data.value.topic.postsCount }} 篇讨论</span>
      </div>
    </section>

    <section class="feed-panel topic-feed-panel">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">相关讨论</button>
        </div>
      </div>
      <div class="feed-grid">
        <PostCard v-for="post in topicQuery.data.value?.posts ?? []" :key="post.id" :post="post" />
      </div>
    </section>
  </main>
</template>
