<script setup lang="ts">
import { computed } from 'vue'
import AuthModal from './components/AuthModal.vue'
import PublishModal from './components/PublishModal.vue'
import TopNav from './components/TopNav.vue'
import { useForumData } from './composables/useForumData'
import { useForumStore } from './stores/forum'
import { useRoute } from 'vue-router'

const forumStore = useForumStore()
const { source } = useForumData()
const route = useRoute()
const isAdminLayout = computed(() => route.meta.layout === 'admin')
</script>

<template>
  <div class="app-shell" :class="{ 'is-home-route': route.path === '/', 'is-admin-route': isAdminLayout }">
    <TopNav v-if="!isAdminLayout" :source="source" />
    <RouterView v-slot="{ Component, route }">
      <Transition name="page-flow" mode="out-in">
        <component :is="Component" :key="route.fullPath" />
      </Transition>
    </RouterView>
    <AuthModal v-if="!isAdminLayout && forumStore.authOpen" />
    <PublishModal v-if="!isAdminLayout && forumStore.publishOpen" />
  </div>
</template>
