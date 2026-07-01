<script setup lang="ts">
import AuthModal from './components/AuthModal.vue'
import PublishModal from './components/PublishModal.vue'
import TopNav from './components/TopNav.vue'
import { useForumData } from './composables/useForumData'
import { useForumStore } from './stores/forum'

const forumStore = useForumStore()
const { source } = useForumData()
</script>

<template>
  <div class="app-shell">
    <TopNav :source="source" />
    <RouterView />
    <AuthModal v-if="forumStore.authOpen" />
    <PublishModal v-if="forumStore.publishOpen" />
  </div>
</template>
