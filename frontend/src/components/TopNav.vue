<script setup lang="ts">
import { Bell, BookOpen, ChevronDown, LogOut, Mail, PenLine, Search, Settings } from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import DecisionSearch from './DecisionSearch.vue'
import { useForumData } from '../composables/useForumData'
import { useForumStore } from '../stores/forum'
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
const { posts, topics } = useForumData()

const navItems: Array<{ label: string; category: Category | 'all' }> = [
  { label: '首页', category: 'all' },
  { label: '经验帖', category: 'experience' },
  { label: '数据建议', category: 'data' },
  { label: '家长提问', category: 'question' },
]

const activeCategory = computed(() => (route.path === '/' ? forumStore.filter.category : ''))

function setCategory(category: Category | 'all') {
  forumStore.browseCategory(category)
}

function togglePanel(panel: 'notifications' | 'profile') {
  openPanel.value = openPanel.value === panel ? null : panel
  if (panel === 'notifications') forumStore.markNotificationsRead()
}

function openSearch() {
  if (searchCloseTimer) window.clearTimeout(searchCloseTimer)
  searchOpen.value = true
}

function closeSearchSoon() {
  if (searchCloseTimer) window.clearTimeout(searchCloseTimer)
  searchCloseTimer = window.setTimeout(() => {
    searchOpen.value = false
  }, 180)
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
        <span v-if="forumStore.notificationUnread" class="notification-dot">{{ forumStore.notificationUnread }}</span>
      </button>
      <RouterLink class="icon-button" aria-label="私信" to="/messages" @click="forumStore.markMessagesRead()">
        <Mail :size="20" />
        <span v-if="forumStore.messageUnread" class="message-dot">{{ forumStore.messageUnread }}</span>
      </RouterLink>
      <button
        v-if="!forumStore.currentUser"
        class="login-button"
        type="button"
        @click="forumStore.authOpen = true"
      >
        登录 / 注册
      </button>
      <button v-else class="profile-button" aria-label="个人中心" @click="togglePanel('profile')">
        <span class="avatar">{{ forumStore.currentUser.nickname.slice(0, 1) }}</span>
        <span class="profile-name">{{ forumStore.currentUser.nickname }}</span>
        <ChevronDown :size="15" />
      </button>

      <div v-if="openPanel === 'notifications'" class="nav-popover">
        <strong>通知</strong>
        <p>你关注的“物理方向组合怎么选”新增 3 条讨论。</p>
        <p>系统建议你完善 MBTI 和目标专业，提高推荐准确度。</p>
        <RouterLink to="/settings" @click="openPanel = null">去完善资料</RouterLink>
      </div>

      <div v-if="openPanel === 'profile'" class="nav-popover profile-popover">
        <div>
          <span class="avatar">{{ forumStore.currentUser?.nickname.slice(0, 1) }}</span>
          <strong>{{ forumStore.currentUser?.nickname }}</strong>
          <small>{{ forumStore.currentUser?.grade }} · {{ forumStore.currentUser?.province }}</small>
        </div>
        <RouterLink to="/settings" @click="openPanel = null">
          <Settings :size="16" /> 个人信息与选科画像
        </RouterLink>
        <button type="button" @click="forumStore.logout(); openPanel = null">
          <LogOut :size="16" /> 退出登录
        </button>
      </div>
    </div>
  </header>
</template>
