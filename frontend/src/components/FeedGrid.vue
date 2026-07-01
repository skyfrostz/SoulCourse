<script setup lang="ts">
import { ChevronDown, RefreshCw } from '@lucide/vue'
import PostCard from './PostCard.vue'
import { useForumStore } from '../stores/forum'
import type { Post } from '../types/forum'

defineProps<{
  posts: Post[]
  isLoading: boolean
}>()

const forumStore = useForumStore()
</script>

<template>
  <section class="feed-panel">
    <div class="feed-toolbar">
      <div class="feed-tabs">
        <button :class="{ active: forumStore.filter.sort === 'recommended' }" @click="forumStore.setSort('recommended')">
          推荐
        </button>
        <button :class="{ active: forumStore.filter.sort === 'latest' }" @click="forumStore.setSort('latest')">
          最新
        </button>
      </div>
      <button
        class="sort-button"
        :class="{ active: forumStore.filter.sort === 'hot' }"
        @click="forumStore.setSort(forumStore.filter.sort === 'hot' ? 'recommended' : 'hot')"
      >
        最热 <ChevronDown :size="15" />
      </button>
      <button class="refresh-button" type="button" @click="forumStore.triggerRefreshHint">
        <RefreshCw :size="14" /> 换一批
      </button>
    </div>

    <p v-if="forumStore.refreshHint" class="refresh-hint">{{ forumStore.refreshHint }}</p>

    <div v-if="isLoading" class="feed-grid">
      <div v-for="item in 4" :key="item" class="post-card skeleton"></div>
    </div>

    <div v-else-if="posts.length" class="feed-grid">
      <PostCard v-for="post in posts" :key="post.id" :post="post" />
    </div>

    <div v-else class="empty-state">
      <h2>没有找到匹配的讨论</h2>
      <p>换一个组合或关键词，看看同学和家长们正在聊什么。</p>
    </div>

    <nav class="pagination-bar" aria-label="帖子分页">
      <button :disabled="forumStore.page <= 1" @click="forumStore.setPage(forumStore.page - 1)">上一页</button>
      <span>第 {{ forumStore.page }} 页</span>
      <button :disabled="posts.length < forumStore.pageSize" @click="forumStore.setPage(forumStore.page + 1)">下一页</button>
    </nav>
  </section>
</template>
