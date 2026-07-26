<script setup lang="ts">
import FeedGrid from '../components/FeedGrid.vue'
import FilterRail from '../components/FilterRail.vue'
import InsightPanel from '../components/InsightPanel.vue'
import { useForumData } from '../composables/useForumData'
import { ref } from 'vue'

const { posts, insights, topics, isLoading } = useForumData()
const railCollapsed = ref(false)
</script>

<template>
  <main class="workspace forum-home-workspace" :class="{ 'filter-collapsed': railCollapsed }">
    <FilterRail :collapsed="railCollapsed" @toggle-collapse="railCollapsed = !railCollapsed" />
    <FeedGrid :posts="posts" :is-loading="isLoading" />
    <InsightPanel :insights="insights" :topics="topics" />
  </main>
</template>
