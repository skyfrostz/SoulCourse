<script setup lang="ts">
import { useInfiniteQuery } from '@tanstack/vue-query'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Bell, ChevronLeft, ExternalLink, Heart, MessageCircle, Plus, RefreshCcw, Search, ShieldCheck, UserPlus, WifiOff } from '@lucide/vue'
import { fetchNotifications } from '../lib/api'
import { notificationTypeLabels } from '../lib/notifications'
import { useForumStore } from '../stores/forum'
import type { NotificationType } from '../types/forum'

const router = useRouter()
const forumStore = useForumStore()
type NotificationFilter = NotificationType | 'all' | 'engagement'
const activeType = ref<NotificationFilter>('all')
const activeId = ref<number | null>(null)
const keyword = ref('')
const mobileSearchOpen = ref(false)
const isOffline = ref(typeof navigator !== 'undefined' ? !navigator.onLine : false)

const typeTabs: Array<{ label: string; value: NotificationFilter }> = [
  { label: '全部', value: 'all' },
  { label: '评论互动', value: 'comment' },
  { label: '赞与收藏', value: 'engagement' },
  { label: '画像建议', value: 'profile' },
  { label: '关注动态', value: 'follow' },
  { label: '系统提醒', value: 'system' },
]

const notificationsQuery = useInfiniteQuery({
  queryKey: ['notifications'],
  queryFn: ({ pageParam }) => fetchNotifications({ limit: 30, cursor: pageParam }),
  initialPageParam: undefined as string | undefined,
  getNextPageParam: (lastPage) => lastPage.hasMore ? lastPage.nextCursor : undefined,
  retry: false,
  enabled: computed(() => forumStore.isAuthed),
})

watch(() => notificationsQuery.data.value, (data) => {
  if (!data) return
  const seen = new Set<number>()
  forumStore.notifications = data.pages.flatMap((page) => page.items).filter((item) => {
    if (seen.has(item.id)) return false
    seen.add(item.id)
    return true
  })
})

function loadMoreNotifications() {
  if (!isOffline.value && notificationsQuery.hasNextPage.value && !notificationsQuery.isFetchingNextPage.value) {
    void notificationsQuery.fetchNextPage()
  }
}

const notifications = computed(() => forumStore.notifications.map((item) => ({ ...item, unread: !item.readAt })))
const initialLoadFailed = computed(() => notificationsQuery.isError.value && notifications.value.length === 0)
const hasSearchFilter = computed(() => keyword.value.trim().length > 0)

const filteredNotifications = computed(() => {
  const q = keyword.value.trim()
  return notifications.value.filter((item) =>
    (activeType.value === 'all' || item.type === activeType.value || (activeType.value === 'engagement' && ['like', 'favorite'].includes(item.type))) &&
    (!q || [item.title, item.summary, notificationTypeLabels[item.type]].some((value) => value.includes(q))),
  )
})

const activeNotification = computed(() =>
  notifications.value.find((item) => item.id === activeId.value) ?? filteredNotifications.value[0],
)

function selectNotification(id: number) {
  activeId.value = id
  void forumStore.markNotificationsRead([id])
}

function openMobileNotification(id: number, targetUrl: string) {
  selectNotification(id)
  router.push(targetUrl)
}

function openTarget() {
  if (!activeNotification.value) return
  void forumStore.markNotificationsRead([activeNotification.value.id])
  router.push(activeNotification.value.targetUrl)
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

function formatTime(value: string) {
  return new Date(value).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatShortTime(value: string) {
  return new Date(value).toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
}
</script>

<template>
  <main class="detail-page notifications-page">
    <section v-if="!forumStore.isAuthed" class="empty-state detail-empty-state public-page-state notification-auth-state">
      <Bell :size="34" />
      <h1>登录后查看通知</h1>
      <p>评论、点赞、收藏、关注和系统提醒会跟随账号同步。</p>
      <button class="primary-wide compact" type="button" @click="forumStore.openAuth('/notifications')">登录 / 注册</button>
    </section>

    <template v-else>
    <header class="mobile-message-header">
      <h1>消息</h1>
      <div>
        <button type="button" :aria-expanded="mobileSearchOpen" aria-label="搜索消息" @click="mobileSearchOpen = !mobileSearchOpen">
          <Search :size="23" />
        </button>
        <RouterLink to="/messages" aria-label="打开私信"><Plus :size="25" /></RouterLink>
      </div>
    </header>

    <label v-if="mobileSearchOpen" class="mobile-message-search">
      <Search :size="17" />
      <input v-model="keyword" type="search" autocomplete="off" placeholder="搜索消息内容" />
    </label>

    <nav class="mobile-message-shortcuts" aria-label="消息快捷入口">
      <button type="button" :class="{ active: activeType === 'engagement' }" @click="activeType = 'engagement'">
        <span class="tone-like"><Heart :size="25" /></span>
        <strong>赞与收藏</strong>
      </button>
      <button type="button" :class="{ active: activeType === 'follow' }" @click="activeType = 'follow'">
        <span class="tone-follow"><UserPlus :size="25" /></span>
        <strong>新增关注</strong>
      </button>
      <button type="button" :class="{ active: activeType === 'comment' }" @click="activeType = 'comment'">
        <span class="tone-comment"><MessageCircle :size="25" /></span>
        <strong>评论回复</strong>
      </button>
    </nav>

    <section class="mobile-notification-list" aria-label="通知列表">
      <div v-if="notificationsQuery.isLoading.value" class="mobile-message-empty">
        <RefreshCcw class="state-spin" :size="28" />
        <strong>正在加载通知</strong>
        <p>请稍等，正在同步你的未读提醒。</p>
      </div>
      <div v-else-if="isOffline || initialLoadFailed" class="mobile-message-empty">
        <WifiOff v-if="isOffline" :size="28" />
        <Bell v-else :size="28" />
        <strong>{{ isOffline ? '当前网络不可用' : '通知加载失败' }}</strong>
        <p>{{ isOffline ? '恢复网络后会继续同步提醒。' : '请稍后重试，或先返回论坛继续浏览。' }}</p>
        <button v-if="!isOffline" class="state-inline-button" type="button" @click="notificationsQuery.refetch()">重试</button>
      </div>
      <template v-else>
        <button v-for="item in filteredNotifications" :key="item.id" type="button" @click="openMobileNotification(item.id, item.targetUrl)">
          <span class="mobile-notification-avatar" :class="`tone-${item.type}`">
            <Heart v-if="['like', 'favorite'].includes(item.type)" :size="21" />
            <MessageCircle v-else-if="item.type === 'comment'" :size="21" />
            <UserPlus v-else-if="item.type === 'follow'" :size="21" />
            <ShieldCheck v-else :size="21" />
          </span>
          <span class="mobile-notification-copy">
            <strong>{{ item.title }}</strong>
            <small>{{ item.summary }}</small>
          </span>
          <time>{{ formatShortTime(item.createdAt) }}</time>
          <i v-if="item.unread"></i>
        </button>
      </template>
      <div v-if="notificationsQuery.hasNextPage.value || notificationsQuery.isFetchingNextPage.value || notificationsQuery.isFetchNextPageError.value" class="mobile-message-empty">
        <p v-if="notificationsQuery.isFetchNextPageError.value" role="alert">更多通知加载失败，已加载内容不受影响。</p>
        <button class="state-inline-button" type="button" :disabled="isOffline || notificationsQuery.isFetchingNextPage.value" @click="loadMoreNotifications">
          {{ notificationsQuery.isFetchingNextPage.value ? '正在加载更多' : notificationsQuery.isFetchNextPageError.value ? '重试加载更多' : '加载更多' }}
        </button>
      </div>
      <div v-if="!notificationsQuery.isLoading.value && !initialLoadFailed && !isOffline && !filteredNotifications.length" class="mobile-message-empty">
        <Bell :size="28" />
        <strong>{{ hasSearchFilter ? '没有匹配的通知' : '暂时没有新消息' }}</strong>
        <p>{{ hasSearchFilter ? '试试清空搜索词，或者换一个通知关键词。' : '与你有关的赞、收藏、关注和评论会出现在这里。' }}</p>
      </div>
    </section>

    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>

    <section class="notifications-hero">
      <div>
        <div class="breadcrumb">个人中心 / 通知</div>
        <h1>通知中心</h1>
        <p>集中查看话题评论、政策库更新、关注动态和画像建议。读过的通知会自动消除顶部红点。</p>
        <div class="overview-metrics">
          <span><Bell :size="17" /> {{ forumStore.unreadNotificationCount }} 条未读</span>
          <span><ShieldCheck :size="17" /> 政策与互动提醒</span>
        </div>
      </div>
      <label class="knowledge-search">
        <Search :size="18" />
        <input v-model="keyword" placeholder="搜索通知内容..." />
      </label>
    </section>

    <nav class="content-lens-tabs notification-tabs" aria-label="通知筛选">
      <RouterLink class="notification-private-link" to="/messages">
        <MessageCircle :size="15" /> 私信
      </RouterLink>
      <button
        v-for="tab in typeTabs"
        :key="tab.value"
        type="button"
        :class="{ active: activeType === tab.value }"
        @click="activeType = tab.value"
      >
        {{ tab.label }}
      </button>
      <button type="button" @click="forumStore.markNotificationsRead()">全部已读</button>
    </nav>
    <p v-if="forumStore.notificationReadError" class="notification-sync-error">{{ forumStore.notificationReadError }}</p>

    <section class="notifications-shell">
      <aside class="notification-list-panel">
        <div v-if="notificationsQuery.isLoading.value" class="empty-state compact-empty notification-panel-state">
          <RefreshCcw class="state-spin" :size="30" />
          <h2>正在加载通知</h2>
          <p>请稍等，正在同步你的未读提醒。</p>
        </div>
        <div v-else-if="isOffline || initialLoadFailed" class="empty-state compact-empty notification-panel-state">
          <WifiOff v-if="isOffline" :size="30" />
          <Bell v-else :size="30" />
          <h2>{{ isOffline ? '当前网络不可用' : '通知加载失败' }}</h2>
          <p>{{ isOffline ? '恢复网络后会继续同步提醒。' : '请稍后重试，或先返回论坛继续浏览。' }}</p>
          <button v-if="!isOffline" class="primary-wide compact" type="button" @click="notificationsQuery.refetch()">重试</button>
        </div>
        <template v-else>
          <button
            v-for="item in filteredNotifications"
            :key="item.id"
            type="button"
            :class="{ active: activeNotification?.id === item.id, unread: item.unread }"
            @click="selectNotification(item.id)"
          >
            <span></span>
            <small>{{ notificationTypeLabels[item.type] }} · {{ formatTime(item.createdAt) }}</small>
            <strong>{{ item.title }}</strong>
            <p>{{ item.summary }}</p>
          </button>
          <div v-if="notificationsQuery.hasNextPage.value || notificationsQuery.isFetchingNextPage.value || notificationsQuery.isFetchNextPageError.value" class="empty-state compact-empty notification-panel-state">
            <p v-if="notificationsQuery.isFetchNextPageError.value" role="alert">更多通知加载失败，已加载内容不受影响。</p>
            <button class="primary-wide compact" type="button" :disabled="isOffline || notificationsQuery.isFetchingNextPage.value" @click="loadMoreNotifications">
              {{ notificationsQuery.isFetchingNextPage.value ? '正在加载更多' : notificationsQuery.isFetchNextPageError.value ? '重试加载更多' : '加载更多' }}
            </button>
          </div>
        </template>
      </aside>

      <article v-if="!notificationsQuery.isLoading.value && !initialLoadFailed && !isOffline && activeNotification" class="notification-detail-panel">
        <small>{{ notificationTypeLabels[activeNotification.type] }} · {{ formatTime(activeNotification.createdAt) }}</small>
        <h2>{{ activeNotification.title }}</h2>
        <p>{{ activeNotification.summary }}</p>
        <div class="notification-detail-actions">
          <button class="primary-wide compact" type="button" @click="openTarget">
            查看相关内容 <ExternalLink :size="15" />
          </button>
          <button type="button" @click="forumStore.markNotificationsRead([activeNotification.id])">标记已读</button>
        </div>
      </article>

      <section v-else-if="!notificationsQuery.isLoading.value && !initialLoadFailed && !isOffline" class="empty-state compact-empty">
        <Bell :size="30" />
        <h2>{{ hasSearchFilter ? '没有匹配的通知' : '暂无通知' }}</h2>
        <p>{{ hasSearchFilter ? '试试清空搜索词，或者换一个通知关键词。' : '有新的评论、政策更新和关注动态时，会出现在这里。' }}</p>
      </section>
    </section>
    </template>
  </main>
</template>
