<script setup lang="ts">
import FeedGrid from '../components/FeedGrid.vue'
import FilterRail from '../components/FilterRail.vue'
import InsightPanel from '../components/InsightPanel.vue'
import { useForumData } from '../composables/useForumData'
import { useGlobalSearch } from '../composables/useGlobalSearch'
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'

const { posts, insights, topics, isLoading } = useForumData()
const railCollapsed = ref(false)
const route = useRoute()
const { syncSearchFromRoute } = useGlobalSearch()

watch(
  () => route.query.q,
  () => syncSearchFromRoute(),
  { immediate: true },
)
</script>

<template>
  <main class="workspace forum-home-workspace" :class="{ 'filter-collapsed': railCollapsed }">
    <FilterRail :collapsed="railCollapsed" :insights="insights" @toggle-collapse="railCollapsed = !railCollapsed" />
    <FeedGrid :posts="posts" :is-loading="isLoading" />
    <InsightPanel :insights="insights" :topics="topics" />
  </main>
</template>
