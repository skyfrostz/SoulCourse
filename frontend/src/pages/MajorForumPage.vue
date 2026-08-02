<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { Bookmark, ChevronLeft, MessageCircle, PenLine, RefreshCcw, Search, ShieldCheck, Sparkles, ThumbsUp, TrendingUp, WifiOff } from '@lucide/vue'
import { useRoute, useRouter } from 'vue-router'
import PostCard from '../components/PostCard.vue'
import { categoryLabels } from '../lib/labels'
import { fetchPostCollection, fetchPublishedRequirements } from '../lib/api'
import { findMajorRequirement, formatCompactCount, getMajorForumStats, hydrateMajorPosts, majorForumPath, toMajorRequirementCard } from '../lib/majorForum'
import { useForumStore } from '../stores/forum'
import { useOnlineState } from '../composables/useOnlineState'
import type { Category, Post } from '../types/forum'

const route = useRoute()
const router = useRouter()
const forumStore = useForumStore()
const { isOffline } = useOnlineState()
const activeCategory = ref<Category | 'all'>('all')
const activeSort = ref<'hot' | 'latest' | 'saved'>('hot')

const majorName = computed(() => decodeURIComponent(String(route.params.major ?? '')))
const requirementsQuery = useQuery({
  queryKey: ['real-data', 'requirements'],
  queryFn: fetchPublishedRequirements,
})
const requirementCards = computed(() => (requirementsQuery.data.value ?? []).map(toMajorRequirementCard))
const requirement = computed(() => findMajorRequirement(majorName.value, requirementCards.value))
const displayMajor = computed(() => requirement.value?.major ?? majorName.value)
const forumPostsQuery = useQuery({
  queryKey: computed(() => ['major-posts', displayMajor.value]),
  queryFn: () => fetchPostCollection({ sort: 'latest', limit: 50 }),
  enabled: computed(() => Boolean(displayMajor.value)),
})
const forumPosts = computed(() => forumPostsQuery.data.value ?? [])
const matchingContext = computed(() => ({ subjects: requirement.value?.requiredSubjects ?? [], category: requirement.value?.category }))
const relatedPosts = computed(() => hydrateMajorPosts(displayMajor.value, forumStore, forumPosts.value, matchingContext.value))
const stats = computed(() => getMajorForumStats(displayMajor.value, forumStore, forumPosts.value, matchingContext.value))
const categoryTabs: Array<{ label: string; value: Category | 'all' }> = [
  { label: '全部', value: 'all' },
  { label: '经验', value: 'experience' },
  { label: '数据', value: 'data' },
  { label: '提问', value: 'question' },
]

const sortedPosts = computed(() => {
  const filtered = activeCategory.value === 'all'
    ? relatedPosts.value
    : relatedPosts.value.filter((post) => post.category === activeCategory.value)
  return [...filtered].sort((a, b) => {
    if (activeSort.value === 'latest') return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    if (activeSort.value === 'saved') return b.favoritesCount - a.favoritesCount
    return b.likesCount + b.commentsCount + b.favoritesCount - (a.likesCount + a.commentsCount + a.favoritesCount)
  })
})

const relatedMajors = computed(() => {
  const current = requirement.value
  const records = requirementCards.value
  if (!current) return records.filter((item) => item.major !== displayMajor.value).slice(0, 8)
  return records
    .filter((item) => item.category === current.category && item.major !== current.major)
    .slice(0, 10)
})

const topPosts = computed(() => sortedPosts.value.slice(0, 3))

function openPublish(category: Category = 'question') {
  forumStore.openPublish(category)
}

function postScore(post: Post) {
  return post.likesCount + post.commentsCount + post.favoritesCount
}
</script>

<template>
  <main class="detail-page major-forum-page">
    <button class="back-link" @click="router.push('/requirements')"><ChevronLeft :size="17" /> 返回选科查询</button>

    <section class="major-forum-hero">
      <div>
        <div class="breadcrumb">专业论坛 / {{ requirement?.category ?? '专业讨论' }}</div>
        <h1>{{ displayMajor }}讨论论坛</h1>
        <p v-if="requirement">
          {{ requirement.risk }} 这里聚合该专业下的数据核对、学长经验和真实提问，点赞、评论、收藏与帖子详情实时一致。
        </p>
        <p v-else>
          这里聚合与“{{ displayMajor }}”相关的论坛讨论。可以从帖子评论区继续追问，也可以发布自己的选科问题。
        </p>
        <div class="overview-metrics major-metrics">
          <span><Sparkles :size="18" /> {{ stats.postCount }} 篇相关帖子</span>
          <span><ThumbsUp :size="18" /> {{ formatCompactCount(stats.likesCount) }} 点赞</span>
          <span><MessageCircle :size="18" /> {{ formatCompactCount(stats.commentsCount) }} 评论</span>
          <span><Bookmark :size="18" /> {{ formatCompactCount(stats.favoritesCount) }} 收藏</span>
        </div>
      </div>
      <aside class="major-requirement-brief">
        <small><ShieldCheck :size="16" /> 选科要求摘要</small>
        <h2>{{ requirement?.requiredSubjects.join(' / ') ?? '等待补充官方目录' }}</h2>
        <p><strong>建议组合：</strong>{{ requirement?.suggestedCombination ?? '先按目标院校目录核对' }}</p>
        <p v-if="isOffline || requirementsQuery.isError.value">
          {{ isOffline ? '当前网络不可用，官方要求尚未同步；社区讨论仍可查看。' : '专业要求接口暂时不可用，本页不生成模拟结论。' }}
        </p>
        <p v-else-if="!requirementsQuery.isLoading.value && !requirement">暂无已复核数据，请以目标院校最新目录为准。</p>
        <a v-if="requirement?.sourceUrl" :href="requirement.sourceUrl" target="_blank" rel="noreferrer">
          查看官方口径
        </a>
      </aside>
    </section>

    <section class="major-forum-toolbar">
      <div class="scroll-chip-row">
        <button
          v-for="tab in categoryTabs"
          :key="tab.value"
          type="button"
          :class="{ active: activeCategory === tab.value }"
          @click="activeCategory = tab.value"
        >
          {{ tab.label }}
        </button>
      </div>
      <div class="major-sort-row">
        <button type="button" :class="{ active: activeSort === 'hot' }" @click="activeSort = 'hot'">
          <TrendingUp :size="15" /> 热度
        </button>
        <button type="button" :class="{ active: activeSort === 'latest' }" @click="activeSort = 'latest'">
          最新
        </button>
        <button type="button" :class="{ active: activeSort === 'saved' }" @click="activeSort = 'saved'">
          <Bookmark :size="15" /> 收藏
        </button>
        <button class="write-button" type="button" @click="openPublish('question')">
          <PenLine :size="16" /> 发起{{ displayMajor }}提问
        </button>
      </div>
    </section>

    <section class="major-forum-layout">
      <div>
        <section v-if="forumPostsQuery.isLoading.value" class="empty-state">
          <RefreshCcw class="state-spin" :size="30" />
          <h2>正在加载相关帖子</h2>
          <p>正在同步 {{ displayMajor }} 的最新讨论。</p>
        </section>
        <section v-else-if="isOffline || forumPostsQuery.isError.value" class="empty-state">
          <WifiOff v-if="isOffline" :size="30" />
          <MessageCircle v-else :size="30" />
          <h2>{{ isOffline ? '当前网络不可用' : '帖子加载失败' }}</h2>
          <p>{{ isOffline ? '恢复网络后可继续查看专业讨论。' : '后端服务或网络暂时不可用，请稍后重试。' }}</p>
          <button v-if="!isOffline" class="state-inline-button" type="button" @click="forumPostsQuery.refetch()">重试</button>
        </section>
        <div v-else-if="sortedPosts.length" class="feed-grid major-forum-feed">
          <PostCard v-for="post in sortedPosts" :key="post.id" :post="post" />
        </div>
        <section v-else class="empty-state">
          <Search :size="30" />
          <h2>暂时没有匹配帖子</h2>
          <p>换一个分类，或直接发布一个与 {{ displayMajor }} 相关的问题。</p>
        </section>
      </div>

      <aside class="major-forum-side">
        <section>
          <strong>本专业高热内容</strong>
          <RouterLink v-for="post in topPosts" :key="post.id" :to="`/posts/${post.id}`">
            <span>{{ categoryLabels[post.category] }}</span>
            <b>{{ post.title }}</b>
            <small>{{ formatCompactCount(postScore(post)) }} 热度 · {{ post.commentsCount }} 评论</small>
          </RouterLink>
        </section>
        <section>
          <strong>同门类继续刷</strong>
          <RouterLink v-for="item in relatedMajors" :key="item.major" :to="majorForumPath(item.major)">
            <span>{{ item.noteType }}</span>
            <b>{{ item.major }}</b>
            <small>{{ item.suggestedCombination }}</small>
          </RouterLink>
        </section>
      </aside>
    </section>
  </main>
</template>
