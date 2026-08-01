<script setup lang="ts">
import { computed, onErrorCaptured, onMounted, watch } from 'vue'
import AuthModal from './components/AuthModal.vue'
import MobileBottomNav from './components/MobileBottomNav.vue'
import PublishModal from './components/PublishModal.vue'
import TopNav from './components/TopNav.vue'
import { useForumData } from './composables/useForumData'
import { appError, clearAppError, showAppError } from './lib/appError'
import { useForumStore } from './stores/forum'
import { useRoute } from 'vue-router'

const forumStore = useForumStore()
const { source } = useForumData()
const route = useRoute()
const isAdminLayout = computed(() => route.meta.layout === 'admin')

onMounted(() => {
  if (forumStore.isAuthed) void forumStore.hydrateAccount()
})

watch(() => route.fullPath, () => clearAppError())

onErrorCaptured((error) => {
  showAppError(error, 'runtime')
  return false
})

function reloadPage() {
  window.location.reload()
}
</script>

<template>
  <div class="app-shell" :class="{ 'is-home-route': route.path === '/', 'is-admin-route': isAdminLayout }">
    <TopNav v-if="!isAdminLayout" :source="source" />
    <main v-if="appError" class="detail-page app-error-page">
      <section class="empty-state detail-empty-state public-page-state">
        <h1>{{ appError.title }}</h1>
        <p>{{ appError.message }}</p>
        <div class="state-action-row">
          <button class="primary-wide compact" type="button" @click="reloadPage">刷新页面</button>
          <RouterLink class="ghost-button compact" to="/">返回首页</RouterLink>
        </div>
      </section>
    </main>
    <RouterView v-else v-slot="{ Component, route }">
      <Transition name="page-flow" mode="out-in">
        <component :is="Component" :key="route.fullPath" />
      </Transition>
    </RouterView>
    <MobileBottomNav v-if="!isAdminLayout" />
    <AuthModal v-if="!isAdminLayout && forumStore.authOpen" />
    <div v-if="!isAdminLayout && forumStore.publishOpen" v-show="!forumStore.authOpen">
      <PublishModal />
    </div>
  </div>
</template>
