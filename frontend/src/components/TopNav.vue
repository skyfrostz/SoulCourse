<script setup lang="ts">
import { Bell, Bookmark, ChevronDown, LogOut, Mail, PenLine, Search, Settings, Users, X } from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import DecisionSearch from './DecisionSearch.vue'
import { useForumData } from '../composables/useForumData'
import { useGlobalSearch } from '../composables/useGlobalSearch'
import { notificationTypeLabels } from '../lib/notifications'
import { appAssetUrl } from '../lib/runtime'
import { useForumStore } from '../stores/forum'
import type { Category } from '../types/forum'

defineProps<{
  source: 'api'
}>()

const forumStore = useForumStore()
const route = useRoute()
const openPanel = ref<'notifications' | 'profile' | null>(null)
const searchOpen = ref(false)
const searchRoot = ref<HTMLElement | null>(null)
let searchCloseTimer: ReturnType<typeof window.setTimeout> | undefined
let panelCloseTimer: ReturnType<typeof window.setTimeout> | undefined
const popoverCloseDelay = 450
const { posts, topics } = useForumData()
const { runSearch } = useGlobalSearch()
const favoritePosts = computed(() => forumStore.getFavoritePosts(posts.value).slice(0, 8))
const notificationItems = computed(() => forumStore.notifications.map((item) => ({ ...item, unread: !item.readAt })))

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
  if (panel === 'notifications' && !forumStore.requireAuth('/notifications')) return
  keepPanelOpen()
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

function keepSearchOpen() {
  if (searchCloseTimer) window.clearTimeout(searchCloseTimer)
}

function closeSearchSoon() {
  if (searchCloseTimer) window.clearTimeout(searchCloseTimer)
  searchCloseTimer = window.setTimeout(() => {
    searchOpen.value = false
  }, popoverCloseDelay)
}

function keepPanelOpen() {
  if (panelCloseTimer) window.clearTimeout(panelCloseTimer)
}

function closePanelSoon(panel: 'notifications' | 'profile') {
  if (panelCloseTimer) window.clearTimeout(panelCloseTimer)
  panelCloseTimer = window.setTimeout(() => {
    if (openPanel.value === panel) openPanel.value = null
  }, popoverCloseDelay)
}

function closeSearchAfterFocus(event: FocusEvent) {
  const nextTarget = event.relatedTarget as Node | null
  if (nextTarget && searchRoot.value?.contains(nextTarget)) return
  closeSearchSoon()
}

function closeSearchWhenOutside(event: PointerEvent) {
  const target = event.target as Node | null
  if (target && searchRoot.value?.contains(target)) return
  searchOpen.value = false
}

async function submitSearch() {
  await runSearch()
  searchOpen.value = false
}

function syncClearedSearch(event: Event) {
  const value = (event.target as HTMLInputElement).value
  if (!value) void runSearch('')
}

onMounted(() => {
  window.addEventListener('pointerdown', closeSearchWhenOutside)
})

onBeforeUnmount(() => {
  if (searchCloseTimer) window.clearTimeout(searchCloseTimer)
  if (panelCloseTimer) window.clearTimeout(panelCloseTimer)
  window.removeEventListener('pointerdown', closeSearchWhenOutside)
})
</script>

<template>
  <header class="top-nav">
    <div class="mobile-discovery-bar">
      <RouterLink class="mobile-brand-mark" to="/" aria-label="选科π首页">
        <img :src="appAssetUrl('/brand/logo-mark.png')" alt="" />
      </RouterLink>
      <nav aria-label="移动端主入口">
        <RouterLink class="active" to="/" aria-current="page" @click="forumStore.resetFilters()">推荐</RouterLink>
        <RouterLink to="/requirements">选科查询</RouterLink>
        <RouterLink to="/knowledge">政策库</RouterLink>
      </nav>
      <button class="mobile-search-button" type="button" :aria-expanded="searchOpen" aria-label="搜索" @pointerdown.stop @click="searchOpen = !searchOpen">
        <X v-if="searchOpen" :size="24" />
        <Search v-else :size="25" />
      </button>
    </div>

    <Transition name="soft-pop">
      <section v-if="searchOpen" class="mobile-search-drawer" @pointerdown.stop>
        <form role="search" @submit.prevent="submitSearch">
          <Search :size="18" />
          <input
            :value="forumStore.filter.keyword"
            type="search"
            autocomplete="off"
            placeholder="搜索组合、专业或经验"
            @input="forumStore.setKeyword(($event.target as HTMLInputElement).value)"
            @search="syncClearedSearch"
          />
          <button type="submit" aria-label="搜索">
            <Search :size="18" />
          </button>
        </form>
        <DecisionSearch :posts="posts" :topics="topics" @searched="searchOpen = false" />
      </section>
    </Transition>

    <RouterLink class="brand-block" to="/" aria-label="选科π首页">
      <div class="brand-mark" aria-hidden="true">
        <img :src="appAssetUrl('/brand/logo-mark.png')" alt="" />
      </div>
      <span class="brand-copy">
        <span class="brand-name">选科π</span>
        <small>广东选科社区</small>
        <span class="brand-spectrum" aria-hidden="true">
          <i></i><i></i><i></i><i></i><i></i><i></i>
        </span>
      </span>
    </RouterLink>

    <nav class="main-tabs" aria-label="主导航">
      <RouterLink
        v-for="item in navItems"
        :key="item.category"
        :class="{ active: activeCategory === item.category }"
        :aria-current="activeCategory === item.category ? 'page' : undefined"
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

    <form ref="searchRoot" class="search-box" role="search" @submit.prevent="submitSearch" @pointerenter="keepSearchOpen" @pointerleave="closeSearchSoon" @focusout="closeSearchAfterFocus">
      <input
        :value="forumStore.filter.keyword"
        type="search"
        placeholder="搜索专业、大学、组合或经验"
        @click="openSearch"
        @input="forumStore.setKeyword(($event.target as HTMLInputElement).value)"
        @search="syncClearedSearch"
      />
      <button type="submit" aria-label="搜索">
        <Search :size="18" />
      </button>
      <Transition name="soft-pop">
        <div v-if="searchOpen" class="search-popover" @pointerenter="keepSearchOpen">
          <button class="search-popover-close" type="button" @click="searchOpen = false">收起</button>
          <DecisionSearch :posts="posts" :topics="topics" @searched="searchOpen = false" />
        </div>
      </Transition>
    </form>

    <div class="nav-actions">
      <button class="write-button" type="button" @click="forumStore.openPublish('question')">
        <PenLine :size="16" /> 发帖
      </button>
      <div
        class="notification-menu-root"
        @pointerenter="keepPanelOpen"
        @pointerleave="closePanelSoon('notifications')"
      >
        <button class="icon-button" aria-label="通知" @click="togglePanel('notifications')">
          <Bell :size="20" />
          <span v-if="forumStore.unreadNotificationCount" class="notification-dot" />
        </button>

        <Transition name="soft-pop">
          <div
            v-if="openPanel === 'notifications'"
            class="nav-popover notification-popover"
            @pointerenter="keepPanelOpen"
          >
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
      <RouterLink class="icon-button" aria-label="私信" to="/messages">
        <Mail :size="20" />
      </RouterLink>
      <button
        v-if="!forumStore.currentUser"
        class="login-button"
        type="button"
        @click="forumStore.openAuth()"
      >
        登录 / 注册
      </button>
      <div
        v-else
        class="profile-menu-root"
        @pointerenter="keepPanelOpen"
        @pointerleave="closePanelSoon('profile')"
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
            @pointerenter="keepPanelOpen"
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

    </div>
  </header>
</template>
