<script setup lang="ts">
import { computed, ref } from 'vue'
import { Bookmark, ChevronLeft, MessageCircle, PenLine, Search, ShieldCheck, Sparkles, ThumbsUp, TrendingUp } from '@lucide/vue'
import { useRoute, useRouter } from 'vue-router'
import PostCard from '../components/PostCard.vue'
import { categoryLabels } from '../lib/labels'
import { majorRequirements } from '../lib/majorRequirements'
import { findMajorRequirement, formatCompactCount, getMajorForumStats, hydrateMajorPosts, majorForumPath } from '../lib/majorForum'
import { samplePosts } from '../lib/sampleData'
import { useForumStore } from '../stores/forum'
import type { Category, Post } from '../types/forum'

const route = useRoute()
const router = useRouter()
const forumStore = useForumStore()
const activeCategory = ref<Category | 'all'>('all')
const activeSort = ref<'hot' | 'latest' | 'saved'>('hot')

const majorName = computed(() => decodeURIComponent(String(route.params.major ?? '')))
const requirement = computed(() => findMajorRequirement(majorName.value))
const displayMajor = computed(() => requirement.value?.major ?? majorName.value)
const forumPosts = computed(() => samplePosts)
const relatedPosts = computed(() => hydrateMajorPosts(displayMajor.value, forumStore, forumPosts.value))
const stats = computed(() => getMajorForumStats(displayMajor.value, forumStore, forumPosts.value))
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
    return b.likesCount + b.commentsCount * 4 + b.favoritesCount * 2 - (a.likesCount + a.commentsCount * 4 + a.favoritesCount * 2)
  })
})

const relatedMajors = computed(() => {
  const current = requirement.value
  if (!current) return majorRequirements.slice(0, 8)
  return majorRequirements
    .filter((item) => item.category === current.category && item.major !== current.major)
    .slice(0, 10)
})

const topPosts = computed(() => sortedPosts.value.slice(0, 3))

function openPublish(category: Category = 'question') {
  forumStore.openPublish(category)
}

function postScore(post: Post) {
  return post.likesCount + post.commentsCount * 4 + post.favoritesCount * 2
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
        <div v-if="sortedPosts.length" class="feed-grid major-forum-feed">
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
