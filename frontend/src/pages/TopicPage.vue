<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { ChevronLeft, Eye, MessageSquare } from '@lucide/vue'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PostCard from '../components/PostCard.vue'
import { apiDataEnabled, fetchTopic } from '../lib/api'
import { useOnlineState } from '../composables/useOnlineState'

const route = useRoute()
const router = useRouter()
const { isOffline } = useOnlineState()
const slug = computed(() => String(route.params.slug))
const topicQuery = useQuery({
  queryKey: computed(() => ['topic-detail', slug.value]),
  queryFn: () => fetchTopic(slug.value),
  enabled: apiDataEnabled,
})
const topicDetail = computed(() => topicQuery.data.value)
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section v-if="topicDetail" class="topic-hero">
      <div class="breadcrumb">首页 / 热门话题</div>
      <h1># {{ topicDetail.topic.title }}</h1>
      <p>{{ topicDetail.topic.summary }}</p>
      <div class="article-actions">
        <span><Eye :size="17" /> {{ topicDetail.topic.viewsCount }} 浏览</span>
        <span><MessageSquare :size="17" /> {{ topicDetail.topic.postsCount }} 篇讨论</span>
      </div>
    </section>

    <section v-else-if="topicQuery.isError.value || isOffline" class="empty-state detail-empty-state">
      <h1>{{ isOffline ? '当前网络不可用' : '话题暂时无法加载' }}</h1>
      <p>请检查网络后重试，本站不会用模拟讨论替代真实内容。</p>
      <button class="primary-wide compact" type="button" @click="topicQuery.refetch()">重新加载</button>
    </section>

    <section v-if="topicQuery.isLoading.value" class="feed-panel topic-feed-panel" aria-label="正在加载讨论">
      <div class="empty-state compact-empty"><p>正在加载相关讨论...</p></div>
    </section>
    <section v-else-if="topicDetail" class="feed-panel topic-feed-panel">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">相关讨论</button>
        </div>
      </div>
      <div class="feed-grid">
        <PostCard v-for="post in topicDetail?.posts ?? []" :key="post.id" :post="post" />
      </div>
      <div v-if="!topicDetail.posts.length" class="empty-state compact-empty">
        <h2>还没有相关讨论</h2>
        <p>成为第一个分享选科经验的人。</p>
      </div>
    </section>
  </main>
</template>
