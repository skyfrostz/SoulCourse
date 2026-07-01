<script setup lang="ts">
import { Sparkles } from '@lucide/vue'
import { computed } from 'vue'
import { majorRequirements } from '../lib/majorRequirements'
import { useForumStore } from '../stores/forum'
import type { Post, Topic } from '../types/forum'

const props = defineProps<{
  posts: Post[]
  topics: Topic[]
}>()

const forumStore = useForumStore()
const keyword = computed(() => forumStore.filter.keyword.trim())
const matchedMajors = computed(() =>
  keyword.value
    ? majorRequirements.filter((item) =>
        [item.major, item.category, item.suggestedCombination, item.requiredSubjects.join('')].some((value) =>
          value.includes(keyword.value),
        ),
      ).slice(0, 3)
    : majorRequirements.slice(0, 3),
)
const matchedTopics = computed(() =>
  props.topics.filter((topic) => !keyword.value || topic.title.includes(keyword.value)).slice(0, 3),
)
const matchedPosts = computed(() =>
  props.posts.filter((post) =>
    !keyword.value ||
    [post.title, post.content, post.authorName, post.tags.join('')].some((value) => value.includes(keyword.value)),
  ).slice(0, 3),
)

const quickQueries = ['临床医学', '计算机', '物化生避坑', '史政地就业', '浙江选科']
</script>

<template>
  <section class="decision-search">
    <div>
      <span><Sparkles :size="16" /> 决策搜索</span>
      <h1>你可能正在找这些</h1>
      <p>搜索会同时覆盖帖子、话题和专业选科要求。</p>
    </div>
    <div class="quick-query-row">
      <button v-for="item in quickQueries" :key="item" type="button" @click="forumStore.setKeyword(item)">
        {{ item }}
      </button>
    </div>
    <div class="decision-result-grid">
      <RouterLink v-for="major in matchedMajors" :key="major.major" to="/requirements">
        <small>专业要求</small>
        <strong>{{ major.major }}</strong>
        <span>{{ major.suggestedCombination }}</span>
      </RouterLink>
      <RouterLink v-for="topic in matchedTopics" :key="topic.slug" :to="`/topics/${topic.slug}`">
        <small>热门话题</small>
        <strong># {{ topic.title }}</strong>
        <span>{{ (topic.viewsCount / 1000).toFixed(1) }}k 浏览</span>
      </RouterLink>
      <RouterLink v-for="post in matchedPosts" :key="post.id" :to="`/posts/${post.id}`">
        <small>经验/数据帖</small>
        <strong>{{ post.title }}</strong>
        <span>{{ post.authorName }} · {{ post.tags.slice(0, 2).join(' / ') || '选科讨论' }}</span>
      </RouterLink>
    </div>
  </section>
</template>
