<script setup lang="ts">
import { Home, MessageCircle, Plus, TrendingUp, UserRound } from '@lucide/vue'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { currentProfileAuthRedirect, useForumStore } from '../stores/forum'

const forumStore = useForumStore()
const route = useRoute()
const router = useRouter()

const unreadCount = computed(() => forumStore.unreadNotificationCount)
const isMessageRoute = computed(() => ['/notifications', '/messages'].includes(route.path))
const isProfileRoute = computed(() =>
  route.path.startsWith('/users/') || ['/settings', '/following'].includes(route.path),
)

function openProfile() {
  if (!forumStore.currentUser) {
    forumStore.openAuth(currentProfileAuthRedirect)
    return
  }
  router.push(`/users/${encodeURIComponent(forumStore.currentUser.nickname)}`)
}
</script>

<template>
  <nav class="mobile-bottom-nav" aria-label="移动端主导航">
    <RouterLink to="/" :class="{ active: route.path === '/' }" aria-label="首页">
      <Home :size="22" />
      <span>首页</span>
    </RouterLink>
    <RouterLink to="/observe" :class="{ active: route.path === '/observe' }" aria-label="广东观察">
      <TrendingUp :size="22" />
      <span>观察</span>
    </RouterLink>
    <button class="mobile-publish-button" type="button" aria-label="发布帖子" @click="forumStore.openPublish('question')">
      <Plus :size="30" />
    </button>
    <RouterLink to="/notifications" :class="{ active: isMessageRoute }" aria-label="消息与通知">
      <span class="mobile-nav-icon">
        <MessageCircle :size="22" />
        <em v-if="unreadCount">{{ unreadCount > 9 ? '9+' : unreadCount }}</em>
      </span>
      <span>消息</span>
    </RouterLink>
    <button type="button" :class="{ active: isProfileRoute }" aria-label="我的主页" @click="openProfile">
      <UserRound :size="22" />
      <span>我的</span>
    </button>
  </nav>
</template>
