<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { ChevronLeft, Heart, PenLine, Search, ShieldCheck, Sparkles } from '@lucide/vue'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import PostCard from '../components/PostCard.vue'
import { fetchPostCollection, fetchTaxonomy } from '../lib/api'
import { useForumStore } from '../stores/forum'

const router = useRouter()
const forumStore = useForumStore()
const keyword = ref('')
const activeTag = ref('')

const taxonomyQuery = useQuery({ queryKey: ['taxonomy'], queryFn: fetchTaxonomy })
const postsQuery = useQuery({
  queryKey: computed(() => ['advice-posts', activeTag.value]),
  queryFn: () => fetchPostCollection({ tag: activeTag.value || undefined, sort: 'latest', limit: 50 }),
})
const subjectTags = computed(() => taxonomyQuery.data.value?.subjectTags ?? [])
const advicePosts = computed(() => {
  const controlled = new Set(subjectTags.value.map((item) => item.value))
  const query = keyword.value.trim().toLowerCase()
  return (postsQuery.data.value ?? []).filter((post) =>
    post.tags.some((tag) => controlled.has(tag)) &&
    (!query || [post.title, post.content, post.authorName, post.province, post.tags.join(' ')].some((value) => value.toLowerCase().includes(query))),
  )
})
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="advice-xhs-hero">
      <div>
        <div class="breadcrumb">首页 / 选科建议</div>
        <h1>同一帖子库里的选科经验</h1>
        <p>所有建议、同组笔记和首页讨论都来自统一帖子库，并按受控选科标签自动归类。</p>
        <div class="overview-metrics">
          <span><Sparkles :size="18" /> {{ advicePosts.length }} 条真实帖子</span>
          <span><Heart :size="18" /> 点赞与评论实时同步</span>
          <span><ShieldCheck :size="18" /> 无匹配标签不强行归类</span>
        </div>
      </div>
      <div class="advice-hero-side">
        <label class="requirement-search">
          <Search :size="18" />
          <input v-model="keyword" placeholder="搜索组合、专业或省份" />
        </label>
        <button class="primary-wide compact" @click="forumStore.openPublish('question')">
          <PenLine :size="16" /> 发布我的选科问题
        </button>
      </div>
    </section>

    <nav class="content-lens-tabs" aria-label="组合筛选">
      <button type="button" :class="{ active: !activeTag }" @click="activeTag = ''">全部组合</button>
      <button
        v-for="tag in subjectTags"
        :key="tag.value"
        type="button"
        :class="{ active: activeTag === tag.value }"
        @click="activeTag = tag.value"
      >
        {{ tag.value }}
      </button>
    </nav>

    <section class="feed-panel topic-feed-panel">
      <div v-if="advicePosts.length" class="feed-grid">
        <PostCard v-for="post in advicePosts" :key="post.id" :post="post" />
      </div>
      <div v-else-if="!postsQuery.isLoading.value" class="empty-state compact-empty">
        <h2>暂无匹配帖子</h2>
        <p>发布后，手动标签和 AI 标签会决定帖子是否进入这里。</p>
      </div>
    </section>
  </main>
</template>
