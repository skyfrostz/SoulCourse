<script setup lang="ts">
import { Sparkles } from '@lucide/vue'
import { computed } from 'vue'
import { useGlobalSearch } from '../composables/useGlobalSearch'
import { provinceKnowledge } from '../lib/knowledgeBase'
import { majorForumPath } from '../lib/majorForum'
import { majorRequirements } from '../lib/majorRequirements'
import { useForumStore } from '../stores/forum'
import type { Post, Topic } from '../types/forum'

const props = defineProps<{
  posts: Post[]
  topics: Topic[]
}>()

const forumStore = useForumStore()
const emit = defineEmits<{
  searched: []
}>()
const { runSearch } = useGlobalSearch()
const keyword = computed(() => forumStore.filter.keyword.trim())
const normalizedKeyword = computed(() => keyword.value.toLowerCase())
const hasMatches = computed(() =>
  matchedMajors.value.length > 0 ||
  matchedTopics.value.length > 0 ||
  matchedPosts.value.length > 0 ||
  matchedPolicies.value.length > 0
)
const matchedMajors = computed(() =>
  normalizedKeyword.value
    ? majorRequirements.filter((item) =>
        [item.major, item.category, item.suggestedCombination, item.requiredSubjects.join('')].some((value) =>
          value.toLowerCase().includes(normalizedKeyword.value),
        ),
      ).slice(0, 3)
    : majorRequirements.slice(0, 3),
)
const matchedTopics = computed(() =>
  props.topics.filter((topic) => !normalizedKeyword.value || topic.title.toLowerCase().includes(normalizedKeyword.value)).slice(0, 3),
)
const matchedPosts = computed(() =>
  props.posts.filter((post) =>
    !normalizedKeyword.value ||
    [post.title, post.content, post.authorName, post.tags.join('')].some((value) => value.toLowerCase().includes(normalizedKeyword.value)),
  ).slice(0, 3),
)
const matchedPolicies = computed(() =>
  provinceKnowledge.filter((item) => {
    if (!normalizedKeyword.value) return true
    const compactKeyword = normalizedKeyword.value.replace(/政策|招生|考试|选科|省份|信息|入口/g, '')
    const fields = [item.province, item.authority, item.status, item.focus.join(''), item.checklist.join('')]
    return fields.some((value) => {
      const normalizedValue = value.toLowerCase()
      return normalizedValue.includes(normalizedKeyword.value) || (!!compactKeyword && normalizedValue.includes(compactKeyword))
    })
  }).slice(0, 2),
)

const quickQueries = ['临床医学', '计算机', '物化生避坑', '史政地就业', '浙江选科', '四川政策']

function provincePath(province: string) {
  return `/knowledge/${encodeURIComponent(province)}`
}

async function searchQuickQuery(query: string) {
  await runSearch(query)
  emit('searched')
}
</script>

<template>
  <section class="decision-search">
    <div>
      <span><Sparkles :size="16" /> 决策搜索</span>
      <h1>你可能正在找这些</h1>
      <p>搜索会同时覆盖帖子、话题和专业选科要求。</p>
    </div>
    <div class="quick-query-row">
      <button v-for="item in quickQueries" :key="item" type="button" @click="searchQuickQuery(item)">
        {{ item }}
      </button>
    </div>
    <div class="decision-result-grid">
      <RouterLink v-for="major in matchedMajors" :key="major.major" :to="majorForumPath(major.major)" @click="emit('searched')">
        <small>专业要求</small>
        <strong>{{ major.major }}</strong>
        <span>{{ major.suggestedCombination }}</span>
      </RouterLink>
      <RouterLink v-for="topic in matchedTopics" :key="topic.slug" :to="`/topics/${topic.slug}`" @click="emit('searched')">
        <small>热门话题</small>
        <strong># {{ topic.title }}</strong>
        <span>{{ (topic.viewsCount / 1000).toFixed(1) }}k 浏览</span>
      </RouterLink>
      <RouterLink v-for="post in matchedPosts" :key="post.id" :to="`/posts/${post.id}`" @click="emit('searched')">
        <small>经验/数据帖</small>
        <strong>{{ post.title }}</strong>
        <span>{{ post.authorName }} · {{ post.tags.slice(0, 2).join(' / ') || '选科讨论' }}</span>
      </RouterLink>
      <RouterLink v-for="item in matchedPolicies" :key="item.province" :to="provincePath(item.province)" @click="emit('searched')">
        <small>省份政策</small>
        <strong>{{ item.province }} · {{ item.reformMode }}</strong>
        <span>{{ item.authority }}</span>
      </RouterLink>
    </div>
    <div v-if="keyword && !hasMatches" class="decision-empty">
      <strong>暂时没有匹配建议</strong>
      <button type="button" @click="searchQuickQuery('')">清除搜索</button>
    </div>
  </section>
</template>
