<script setup lang="ts">
import InsightPanel from '../components/InsightPanel.vue'
import { useForumData } from '../composables/useForumData'

const { insights, topics } = useForumData()
</script>

<template>
  <main class="mobile-observation-page">
    <InsightPanel :insights="insights" :topics="topics" />
  </main>
</template>
