<script setup lang="ts">
import { ChevronDown, MapPin, RefreshCw, Users, WifiOff } from '@lucide/vue'
import { computed } from 'vue'
import PostCard from './PostCard.vue'
import { useGlobalSearch } from '../composables/useGlobalSearch'
import { useForumStore } from '../stores/forum'
import type { Post } from '../types/forum'

defineProps<{
  posts: Post[]
  isLoading: boolean
  hasMore: boolean
  hasError?: boolean
  isOffline?: boolean
}>()

defineEmits<{
  retry: []
}>()

const forumStore = useForumStore()
const { runSearch } = useGlobalSearch()
const activeKeyword = computed(() => forumStore.filter.keyword.trim())
</script>

<template>
  <section class="feed-panel">
    <header class="feed-intro">
      <div>
        <span class="feed-kicker"><MapPin :size="14" /> 广东 · 新高考</span>
        <h1>选科讨论广场</h1>
        <p>学生、家长和老师正在分享真实选择与复盘。</p>
      </div>
      <span class="feed-count"><Users :size="16" /> 本页 {{ posts.length }} 篇讨论</span>
    </header>

    <div class="feed-toolbar">
      <div class="feed-tabs">
        <button :class="{ active: forumStore.filter.sort === 'recommended' }" :aria-pressed="forumStore.filter.sort === 'recommended'" @click="forumStore.setSort('recommended')">
          推荐
        </button>
        <button :class="{ active: forumStore.filter.sort === 'latest' }" :aria-pressed="forumStore.filter.sort === 'latest'" @click="forumStore.setSort('latest')">
          最新
        </button>
      </div>
      <button
        class="sort-button"
        :class="{ active: forumStore.filter.sort === 'hot' }"
        :aria-pressed="forumStore.filter.sort === 'hot'"
        @click="forumStore.setSort(forumStore.filter.sort === 'hot' ? 'recommended' : 'hot')"
      >
        最热 <ChevronDown :size="15" />
      </button>
      <button class="refresh-button" type="button" @click="$emit('retry')">
        <RefreshCw :size="14" /> 换一批
      </button>
    </div>

    <div v-if="activeKeyword" class="search-result-banner">
      <span>正在搜索“{{ activeKeyword }}”</span>
      <button type="button" @click="runSearch('')">清除搜索</button>
    </div>

    <div v-if="isLoading" class="feed-grid">
      <div v-for="item in 4" :key="item" class="post-card skeleton"></div>
    </div>

    <div v-else-if="hasError" class="empty-state">
      <WifiOff v-if="isOffline" :size="28" />
      <RefreshCw v-else :size="28" />
      <h2>{{ isOffline ? '当前网络不可用' : '讨论加载失败' }}</h2>
      <p>{{ isOffline ? '恢复网络后再重试，已发布讨论不会丢失。' : '服务暂时不可用，稍后重试即可继续浏览。' }}</p>
      <button class="primary-wide compact" type="button" @click="$emit('retry')">重试</button>
    </div>

    <div v-else-if="posts.length" class="feed-grid">
      <PostCard v-for="post in posts" :key="post.id" :post="post" />
    </div>

    <div v-else class="empty-state">
      <h2>{{ activeKeyword ? `没有找到“${activeKeyword}”相关讨论` : '没有找到匹配的讨论' }}</h2>
      <p>{{ activeKeyword ? '换一个组合、专业或省份关键词试试。' : '换一个组合或关键词，看看同学和家长们正在聊什么。' }}</p>
      <button v-if="activeKeyword" class="primary-wide compact" type="button" @click="runSearch('')">清除搜索</button>
    </div>

    <nav class="pagination-bar" aria-label="帖子分页">
      <button :disabled="forumStore.page <= 1" @click="forumStore.setPage(forumStore.page - 1)">上一页</button>
      <span>第 {{ forumStore.page }} 页</span>
      <button :disabled="!hasMore" @click="forumStore.setPage(forumStore.page + 1)">下一页</button>
    </nav>
  </section>
</template>
