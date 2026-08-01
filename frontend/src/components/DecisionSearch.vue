<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { Sparkles } from '@lucide/vue'
import { computed } from 'vue'
import { useGlobalSearch } from '../composables/useGlobalSearch'
import { fetchProvinceCoverage, fetchPublishedRequirements } from '../lib/api'
import { majorForumPath, toMajorRequirementCard } from '../lib/majorForum'
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
const requirementsQuery = useQuery({
  queryKey: ['real-data', 'requirements', 'decision-search'],
  queryFn: fetchPublishedRequirements,
})
const provincesQuery = useQuery({
  queryKey: ['real-data', 'provinces', 'decision-search'],
  queryFn: fetchProvinceCoverage,
})
const keyword = computed(() => forumStore.filter.keyword.trim())
const normalizedKeyword = computed(() => keyword.value.toLowerCase())
const requirementCards = computed(() => (requirementsQuery.data.value ?? []).map(toMajorRequirementCard))
const provinceCoverage = computed(() => provincesQuery.data.value ?? [])
const dataSourcesSettled = computed(() => !requirementsQuery.isPending.value && !provincesQuery.isPending.value)
const dataSourcesHealthy = computed(() => !requirementsQuery.isError.value && !provincesQuery.isError.value)
const hasMatches = computed(() =>
  matchedMajors.value.length > 0 ||
  matchedTopics.value.length > 0 ||
  matchedPosts.value.length > 0 ||
  matchedPolicies.value.length > 0
)
const matchedMajors = computed(() =>
  normalizedKeyword.value
    ? requirementCards.value.filter((item) =>
        [item.major, item.category, item.suggestedCombination, item.requiredSubjects.join('')].some((value) =>
          value.toLowerCase().includes(normalizedKeyword.value),
        ),
      ).slice(0, 3)
    : requirementCards.value.slice(0, 3),
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
  provinceCoverage.value.filter((item) => {
    if (!normalizedKeyword.value) return true
    const compactKeyword = normalizedKeyword.value.replace(/政策|招生|考试|选科|省份|信息|入口/g, '')
    const fields = [item.province, item.coverageStatus, String(item.dataYear), item.methodology]
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
        <strong>{{ item.province }} · {{ item.coverageStatus === 'verified' ? '已复核' : '暂无已复核数据' }}</strong>
        <span>{{ item.dataYear }} · {{ item.methodology }}</span>
      </RouterLink>
    </div>
    <div v-if="requirementsQuery.isPending.value" class="decision-source-state" role="status">
      正在加载专业要求…
    </div>
    <div v-else-if="requirementsQuery.isError.value" class="decision-source-state decision-source-error" role="alert">
      <span>专业要求暂时加载失败，其他搜索结果仍可使用。</span>
      <button type="button" :disabled="requirementsQuery.isFetching.value" @click="requirementsQuery.refetch()">
        {{ requirementsQuery.isFetching.value ? '正在重试…' : '重试专业要求' }}
      </button>
    </div>
    <div v-if="provincesQuery.isPending.value" class="decision-source-state" role="status">
      正在加载省份政策…
    </div>
    <div v-else-if="provincesQuery.isError.value" class="decision-source-state decision-source-error" role="alert">
      <span>省份政策暂时加载失败，其他搜索结果仍可使用。</span>
      <button type="button" :disabled="provincesQuery.isFetching.value" @click="provincesQuery.refetch()">
        {{ provincesQuery.isFetching.value ? '正在重试…' : '重试省份政策' }}
      </button>
    </div>
    <div v-if="keyword && !hasMatches && dataSourcesSettled && dataSourcesHealthy" class="decision-empty">
      <strong>暂时没有匹配建议</strong>
      <button type="button" @click="searchQuickQuery('')">清除搜索</button>
    </div>
  </section>
</template>

<style scoped>
.decision-source-state {
  display: flex;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid var(--border-color, #d8dee8);
  border-radius: 6px;
  color: var(--text-secondary, #586174);
  font-size: 14px;
}

.decision-source-error {
  border-color: var(--warning-border, #d99b35);
  color: var(--text-primary, #202534);
}

.decision-source-state button {
  min-height: 36px;
  flex: 0 0 auto;
  border: 0;
  background: transparent;
  color: var(--accent-color, #2367d1);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}

.decision-source-state button:disabled {
  cursor: wait;
  opacity: 0.65;
}

@media (max-width: 480px) {
  .decision-source-state {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
