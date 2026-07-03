<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Bell, ChevronLeft, ExternalLink, Search, ShieldCheck } from '@lucide/vue'
import { notificationSeeds, notificationTypeLabels, type NotificationType } from '../lib/notifications'
import { useForumStore } from '../stores/forum'

const router = useRouter()
const forumStore = useForumStore()
const activeType = ref<NotificationType | 'all'>('all')
const activeId = ref(notificationSeeds[0]?.id ?? '')
const keyword = ref('')

const typeTabs: Array<{ label: string; value: NotificationType | 'all' }> = [
  { label: '全部', value: 'all' },
  { label: '评论互动', value: 'comment' },
  { label: '政策更新', value: 'policy' },
  { label: '画像建议', value: 'profile' },
  { label: '关注动态', value: 'follow' },
  { label: '系统提醒', value: 'system' },
]

const notifications = computed(() =>
  notificationSeeds.map((item) => ({
    ...item,
    unread: !forumStore.readNotificationIds[item.id],
  })),
)

const filteredNotifications = computed(() => {
  const q = keyword.value.trim()
  return notifications.value.filter((item) =>
    (activeType.value === 'all' || item.type === activeType.value) &&
    (!q || [item.title, item.summary, item.body, notificationTypeLabels[item.type]].some((value) => value.includes(q))),
  )
})

const activeNotification = computed(() =>
  notifications.value.find((item) => item.id === activeId.value) ?? filteredNotifications.value[0] ?? notifications.value[0],
)

function selectNotification(id: string) {
  activeId.value = id
  forumStore.markNotificationsRead([id])
}

function openTarget() {
  if (!activeNotification.value) return
  forumStore.markNotificationsRead([activeNotification.value.id])
  router.push(activeNotification.value.targetUrl)
}

function formatTime(value: string) {
  return new Date(value).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<template>
  <main class="detail-page notifications-page">
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

    <section class="notifications-shell">
      <aside class="notification-list-panel">
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
      </aside>

      <article v-if="activeNotification" class="notification-detail-panel">
        <small>{{ notificationTypeLabels[activeNotification.type] }} · {{ formatTime(activeNotification.createdAt) }}</small>
        <h2>{{ activeNotification.title }}</h2>
        <p>{{ activeNotification.body }}</p>
        <div class="notification-detail-actions">
          <button class="primary-wide compact" type="button" @click="openTarget">
            {{ activeNotification.targetLabel }} <ExternalLink :size="15" />
          </button>
          <button type="button" @click="forumStore.markNotificationsRead([activeNotification.id])">标记已读</button>
        </div>
      </article>

      <section v-else class="empty-state compact-empty">
        <Bell :size="30" />
        <h2>暂无通知</h2>
        <p>有新的评论、政策更新和关注动态时，会出现在这里。</p>
      </section>
    </section>
  </main>
</template>
