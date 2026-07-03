<script setup lang="ts">
import { Bell, BookOpen, Bookmark, ChevronDown, LogOut, Mail, PenLine, Search, Settings, Users } from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import DecisionSearch from './DecisionSearch.vue'
import { useForumData } from '../composables/useForumData'
import { notificationSeeds, notificationTypeLabels } from '../lib/notifications'
import { useForumStore } from '../stores/forum'
import { samplePosts } from '../lib/sampleData'
import type { Category } from '../types/forum'

defineProps<{
  source: 'api' | 'local'
}>()

const forumStore = useForumStore()
const route = useRoute()
const openPanel = ref<'notifications' | 'profile' | null>(null)
const searchOpen = ref(false)
const searchRoot = ref<HTMLElement | null>(null)
let searchCloseTimer: ReturnType<typeof window.setTimeout> | undefined
let profileCloseTimer: ReturnType<typeof window.setTimeout> | undefined
const { posts, topics } = useForumData()
const favoritePosts = computed(() => forumStore.getFavoritePosts([...forumStore.getCreatedPosts(), ...samplePosts, ...posts.value]).slice(0, 8))
const notificationItems = computed(() =>
  notificationSeeds.map((item) => ({
    ...item,
    unread: !forumStore.readNotificationIds[item.id],
  })),
)

const navItems: Array<{ label: string; category: Category | 'all' }> = [
  { label: '首页', category: 'all' },
  { label: '经验帖', category: 'experience' },
  { label: '数据建议', category: 'data' },
  { label: '提问', category: 'question' },
]

const activeCategory = computed(() => (route.path === '/' ? forumStore.filter.category : ''))

function setCategory(category: Category | 'all') {
  forumStore.browseCategory(category)
}

function togglePanel(panel: 'notifications' | 'profile') {
  if (profileCloseTimer) window.clearTimeout(profileCloseTimer)
  const willOpen = openPanel.value !== panel
  openPanel.value = willOpen ? panel : null
  if (panel === 'notifications' && willOpen) forumStore.markNotificationsRead()
}

function formatNotificationTime(value: string) {
  return new Date(value).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function openSearch() {
  if (searchCloseTimer) window.clearTimeout(searchCloseTimer)
  searchOpen.value = true
}

function closeSearchSoon() {
  if (searchCloseTimer) window.clearTimeout(searchCloseTimer)
  searchCloseTimer = window.setTimeout(() => {
    searchOpen.value = false
  }, 340)
}

function keepProfilePanelOpen() {
  if (profileCloseTimer) window.clearTimeout(profileCloseTimer)
}

function closeProfileSoon() {
  if (profileCloseTimer) window.clearTimeout(profileCloseTimer)
  profileCloseTimer = window.setTimeout(() => {
    if (openPanel.value === 'profile') openPanel.value = null
  }, 340)
}

function closeSearchWhenOutside(event: PointerEvent) {
  const target = event.target as Node | null
  if (target && searchRoot.value?.contains(target)) return
  searchOpen.value = false
}

onMounted(() => {
  window.addEventListener('pointerdown', closeSearchWhenOutside)
})

onBeforeUnmount(() => {
  if (searchCloseTimer) window.clearTimeout(searchCloseTimer)
  if (profileCloseTimer) window.clearTimeout(profileCloseTimer)
  window.removeEventListener('pointerdown', closeSearchWhenOutside)
})
</script>

<template>
  <header class="top-nav">
    <div class="brand-block">
      <div class="brand-mark" aria-hidden="true">
        <BookOpen :size="28" :stroke-width="2.4" />
      </div>
      <span class="brand-name">选科知谈</span>
    </div>

    <nav class="main-tabs" aria-label="主导航">
      <RouterLink
        v-for="item in navItems"
        :key="item.category"
        :class="{ active: activeCategory === item.category }"
        to="/"
        @click="setCategory(item.category)"
      >
        {{ item.label }}
      </RouterLink>
    </nav>

    <div class="tool-links">
      <RouterLink class="tool-link" to="/requirements">选科查询</RouterLink>
      <RouterLink class="tool-link" to="/knowledge">政策库</RouterLink>
    </div>

    <div ref="searchRoot" class="search-box" @pointerenter="openSearch" @pointerleave="closeSearchSoon" @focusout="closeSearchSoon">
      <input
        :value="forumStore.filter.keyword"
        type="search"
        placeholder="搜索专业、大学、组合或经验"
        @focus="openSearch"
        @input="openSearch(); forumStore.setKeyword(($event.target as HTMLInputElement).value)"
      />
      <Search :size="18" />
      <Transition name="soft-pop">
        <div v-if="searchOpen" class="search-popover" @pointerenter="openSearch" @pointerleave="closeSearchSoon">
          <button class="search-popover-close" type="button" @click="searchOpen = false">收起</button>
          <DecisionSearch :posts="posts" :topics="topics" />
        </div>
      </Transition>
    </div>

    <div class="nav-actions">
      <span class="source-dot" :class="source">{{ source === 'api' ? 'API' : '本地' }}</span>
      <button class="write-button" type="button" @click="forumStore.openPublish('question')">
        <PenLine :size="16" /> 发帖
      </button>
      <button class="icon-button" aria-label="通知" @click="togglePanel('notifications')">
        <Bell :size="20" />
        <span v-if="forumStore.unreadNotificationCount" class="notification-dot" />
      </button>
      <RouterLink class="icon-button" aria-label="私信" to="/messages" @click="forumStore.markMessagesRead()">
        <Mail :size="20" />
        <span v-if="forumStore.messageUnread" class="message-dot" />
      </RouterLink>
      <button
        v-if="!forumStore.currentUser"
        class="login-button"
        type="button"
        @click="forumStore.authOpen = true"
      >
        登录 / 注册
      </button>
      <div
        v-else
        class="profile-menu-root"
        @pointerenter="keepProfilePanelOpen"
        @pointerleave="closeProfileSoon"
      >
        <button class="profile-button" aria-label="个人中心" @click="togglePanel('profile')">
          <span class="avatar">{{ forumStore.currentUser.nickname.slice(0, 1) }}</span>
          <span class="profile-name">{{ forumStore.currentUser.nickname }}</span>
          <ChevronDown :size="15" />
        </button>

        <Transition name="soft-pop">
          <div
            v-if="openPanel === 'profile'"
            class="nav-popover profile-popover"
            @pointerenter="keepProfilePanelOpen"
            @pointerleave="closeProfileSoon"
          >
            <div>
              <span class="avatar">{{ forumStore.currentUser?.nickname.slice(0, 1) }}</span>
              <strong>{{ forumStore.currentUser?.nickname }}</strong>
              <small>{{ forumStore.currentUser?.grade }} · {{ forumStore.currentUser?.province }}</small>
            </div>
            <RouterLink to="/settings" @click="openPanel = null">
              <Settings :size="16" /> 个人信息与选科画像
            </RouterLink>
            <RouterLink
              v-if="forumStore.currentUser"
              :to="`/users/${encodeURIComponent(forumStore.currentUser.nickname)}`"
              @click="openPanel = null"
            >
              <Bookmark :size="16" /> 我的主页与收藏
            </RouterLink>
            <RouterLink to="/following" @click="openPanel = null">
              <Users :size="16" /> 我的关注
            </RouterLink>
            <section class="profile-favorites">
              <strong><Bookmark :size="15" /> 我的收藏</strong>
              <p v-if="!favoritePosts.length">还没有收藏帖子，点进帖子详情收藏后会出现在这里。</p>
              <RouterLink
                v-for="post in favoritePosts"
                :key="post.id"
                :to="`/posts/${post.id}`"
                @click="openPanel = null"
              >
                {{ post.title }}
              </RouterLink>
            </section>
            <button type="button" @click="forumStore.logout(); openPanel = null">
              <LogOut :size="16" /> 退出登录
            </button>
          </div>
        </Transition>
      </div>

      <Transition name="soft-pop">
        <div v-if="openPanel === 'notifications'" class="nav-popover notification-popover">
          <header>
            <strong>通知</strong>
            <RouterLink to="/notifications" @click="openPanel = null">通知中心</RouterLink>
          </header>
          <RouterLink
            v-for="item in notificationItems.slice(0, 4)"
            :key="item.id"
            class="notification-preview-card"
            :to="item.targetUrl"
            @click="openPanel = null; forumStore.markNotificationsRead([item.id])"
          >
            <span :class="{ unread: item.unread }"></span>
            <small>{{ notificationTypeLabels[item.type] }} · {{ formatNotificationTime(item.createdAt) }}</small>
            <strong>{{ item.title }}</strong>
            <p>{{ item.summary }}</p>
          </RouterLink>
          <RouterLink class="notification-center-link" to="/notifications" @click="openPanel = null">
            查看全部通知
          </RouterLink>
        </div>
      </Transition>
    </div>
  </header>
</template>
