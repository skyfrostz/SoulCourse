<script setup lang="ts">
import AuthModal from './components/AuthModal.vue'
import PublishModal from './components/PublishModal.vue'
import TopNav from './components/TopNav.vue'
import { useForumData } from './composables/useForumData'
import { useForumStore } from './stores/forum'
import { useRoute } from 'vue-router'

const forumStore = useForumStore()
const { source } = useForumData()
const route = useRoute()
</script>

<template>
  <div class="app-shell" :class="{ 'is-home-route': route.path === '/' }">
    <TopNav :source="source" />
    <RouterView v-slot="{ Component, route }">
      <Transition name="page-flow" mode="out-in">
        <component :is="Component" :key="route.fullPath" />
      </Transition>
    </RouterView>
    <AuthModal v-if="forumStore.authOpen" />
    <PublishModal v-if="forumStore.publishOpen" />
  </div>
</template>
