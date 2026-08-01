<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ChevronLeft, RefreshCcw, Search, UserPlus, Users, WifiOff } from '@lucide/vue'
import { fetchMyProfile } from '../lib/api'
import { roleLabels } from '../lib/labels'
import { useForumStore } from '../stores/forum'
import type { FollowProfile } from '../types/forum'

const router = useRouter()
const forumStore = useForumStore()
const activeTab = ref<'following' | 'followers'>('following')
const keyword = ref('')
const isOffline = ref(typeof navigator !== 'undefined' ? !navigator.onLine : false)

const profileQuery = useQuery({
  queryKey: ['profile', 'me'],
  queryFn: fetchMyProfile,
  enabled: computed(() => forumStore.isAuthed),
})
const followingList = computed(() => profileQuery.data.value?.following ?? [])
const followerList = computed(() => profileQuery.data.value?.followers ?? [])
const activeList = computed(() => activeTab.value === 'following' ? followingList.value : followerList.value)
const hasSearchFilter = computed(() => keyword.value.trim().length > 0)
const filteredUsers = computed(() => {
  const q = keyword.value.trim()
  if (!q) return activeList.value
  return activeList.value.filter((user) =>
    [user.name, user.province, user.grade, roleLabels[user.role]].some((value) => value.includes(q)),
  )
})

function userPath(user: FollowProfile) {
  return `/users/${encodeURIComponent(user.name)}`
}

function updateOnlineState() {
  isOffline.value = typeof navigator !== 'undefined' ? !navigator.onLine : false
}

onMounted(() => {
  window.addEventListener('online', updateOnlineState)
  window.addEventListener('offline', updateOnlineState)
})

onBeforeUnmount(() => {
  window.removeEventListener('online', updateOnlineState)
  window.removeEventListener('offline', updateOnlineState)
})
</script>

<template>
  <main class="detail-page following-page">
    <button class="back-link" @click="router.back()"><ChevronLeft :size="17" /> 返回上一页</button>

    <section class="following-hero">
      <div>
        <div class="breadcrumb">个人中心 / 关注关系</div>
        <h1>我的关注</h1>
        <p>把你关注的老师、规划师、学长学姐和同组选科同学集中管理，后续可以从这里快速进入主页继续追问。</p>
        <div class="overview-metrics">
          <span><UserPlus :size="17" /> {{ followingList.length }} 个关注</span>
          <span><Users :size="17" /> {{ followerList.length }} 个粉丝</span>
        </div>
      </div>
      <label class="knowledge-search">
        <Search :size="18" />
        <input v-model="keyword" placeholder="搜索姓名、身份、省份..." />
      </label>
    </section>

    <nav class="content-lens-tabs following-tabs" aria-label="关注筛选">
      <button type="button" :class="{ active: activeTab === 'following' }" @click="activeTab = 'following'">
        我的关注 {{ followingList.length }}
      </button>
      <button type="button" :class="{ active: activeTab === 'followers' }" @click="activeTab = 'followers'">
        关注我的 {{ followerList.length }}
      </button>
    </nav>

    <section v-if="forumStore.currentUser && profileQuery.isLoading.value" class="empty-state compact-empty public-page-state">
      <RefreshCcw class="state-spin" :size="30" />
      <h2>正在加载关注关系</h2>
      <p>请稍等，正在同步你的关注和粉丝列表。</p>
    </section>

    <section v-else-if="forumStore.currentUser && (profileQuery.isError.value || isOffline)" class="empty-state compact-empty public-page-state">
      <WifiOff v-if="isOffline" :size="30" />
      <Users v-else :size="30" />
      <h2>{{ isOffline ? '当前网络不可用' : '关注关系加载失败' }}</h2>
      <p>{{ isOffline ? '恢复网络后再刷新页面，关注关系不会丢失。' : '请稍后重试，或返回论坛继续浏览。' }}</p>
      <button v-if="!isOffline" class="primary-wide compact" type="button" @click="profileQuery.refetch()">重试</button>
    </section>

    <section v-else-if="forumStore.currentUser" class="following-user-grid">
      <RouterLink v-for="user in filteredUsers" :key="user.name" class="following-user-card" :to="userPath(user)">
        <span class="user-profile-avatar compact">{{ user.name.slice(0, 1) }}</span>
        <span>
          <strong>{{ user.name }}</strong>
          <small>{{ user.grade }} · {{ roleLabels[user.role] }} · {{ user.province }}</small>
        </span>
        <em>{{ activeTab === 'following' ? '进入主页' : '查看粉丝' }}</em>
      </RouterLink>
    </section>

    <section v-if="forumStore.currentUser && !profileQuery.isLoading.value && !profileQuery.isError.value && !isOffline && !filteredUsers.length" class="empty-state compact-empty">
      <Users :size="30" />
      <h2>
        {{
          hasSearchFilter
            ? (activeTab === 'following' ? '没有匹配的关注用户' : '没有匹配的粉丝')
            : (activeTab === 'following' ? '还没有关注用户' : '还没有人关注你')
        }}
      </h2>
      <p>
        {{
          hasSearchFilter
            ? '试试清空搜索词，或者换一个姓名、省份、身份关键词。'
            : (activeTab === 'following' ? '在用户主页或帖子作者区点击关注，会出现在这里。' : '多发布经验帖、认真回复评论，更容易获得关注。')
        }}
      </p>
    </section>

    <section v-if="!forumStore.currentUser" class="empty-state compact-empty">
      <Users :size="30" />
      <h2>登录后查看关注关系</h2>
      <p>关注、粉丝和收藏会和你的账号一起保存。</p>
      <button class="primary-wide compact" type="button" @click="forumStore.authOpen = true">登录 / 注册</button>
    </section>
  </main>
</template>
