<script setup lang="ts">
import FeedGrid from '../components/FeedGrid.vue'
import FilterRail from '../components/FilterRail.vue'
import InsightPanel from '../components/InsightPanel.vue'
import { useForumData } from '../composables/useForumData'

const { posts, insights, topics, isLoading } = useForumData()
</script>

<template>
  <main class="workspace forum-home-workspace">
    <FilterRail />
    <FeedGrid :posts="posts" :is-loading="isLoading" />
    <InsightPanel :insights="insights" :topics="topics" />
  </main>
</template>
