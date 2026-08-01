<script setup lang="ts">
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { ChevronLeft, MessageCircle, RefreshCcw, Search, Send, WifiOff } from '@lucide/vue'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { fetchConversations, fetchDirectMessages, sendDirectMessage } from '../lib/api'
import { useForumStore } from '../stores/forum'

const route = useRoute()
const router = useRouter()
const forumStore = useForumStore()
const queryClient = useQueryClient()
const draft = ref('')
const sendError = ref('')
const keyword = ref('')
const thread = ref<HTMLElement | null>(null)
const preservingHistoryScroll = ref(false)
const previousThreadHeight = ref(0)
const isOffline = ref(typeof navigator !== 'undefined' ? !navigator.onLine : false)

const requestedPeer = computed(() => String(route.query.to ?? '').trim())
const conversationsQuery = useInfiniteQuery({
  queryKey: ['conversations'],
  queryFn: ({ pageParam }) => fetchConversations({ limit: 30, cursor: pageParam }),
  initialPageParam: undefined as string | undefined,
  getNextPageParam: (lastPage) => lastPage.hasMore ? lastPage.nextCursor : undefined,
  enabled: computed(() => forumStore.isAuthed),
  retry: false,
})
const conversations = computed(() => {
  const seen = new Set<number>()
  return (conversationsQuery.data.value?.pages ?? []).flatMap((page) => page.items).filter((item) => {
    if (seen.has(item.user.id)) return false
    seen.add(item.user.id)
    return true
  })
})
const activePeer = computed(() => requestedPeer.value || conversations.value[0]?.user.nickname || '')
const filteredConversations = computed(() => {
  const q = keyword.value.trim()
  return q ? conversations.value.filter((item) => [item.user.nickname, item.lastMessage].some((value) => value.includes(q))) : conversations.value
})
const activeConversation = computed(() => conversations.value.find((item) => item.user.nickname === activePeer.value))
const messagesQuery = useInfiniteQuery({
  queryKey: computed(() => ['direct-messages', activePeer.value]),
  queryFn: ({ pageParam }) => fetchDirectMessages(activePeer.value, { limit: 50, cursor: pageParam }),
  initialPageParam: undefined as string | undefined,
  getNextPageParam: (lastPage) => lastPage.hasMore ? lastPage.nextCursor : undefined,
  enabled: computed(() => forumStore.isAuthed && Boolean(activePeer.value)),
  retry: false,
})
const messages = computed(() => (messagesQuery.data.value?.pages ?? []).slice().reverse().flatMap((page) => page.items))
const conversationsEmptyTitle = computed(() => keyword.value.trim() ? '没有匹配会话' : '还没有会话')
const conversationsEmptyCopy = computed(() => keyword.value.trim() ? '换个关键词试试，或从用户主页重新发起私信。' : '从真实用户主页发起私信后，会话会保存在这里。')

const sendMutation = useMutation({
  mutationFn: ({ peer, content }: { peer: string; content: string }) => sendDirectMessage(peer, content),
  onSuccess: async (_, variables) => {
    if (activePeer.value === variables.peer) draft.value = ''
    sendError.value = ''
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['conversations'] }),
      queryClient.invalidateQueries({ queryKey: ['direct-messages', variables.peer] }),
    ])
  },
  onError: () => {
    sendError.value = '发送失败，草稿已保留。请确认对方账号存在，或稍后重试。'
  },
})

watch(activePeer, (peer, previousPeer) => {
  if (previousPeer && peer !== previousPeer) {
    draft.value = ''
    sendError.value = ''
  }
})

watch(messages, async () => {
  await nextTick()
  if (!thread.value) return
  if (preservingHistoryScroll.value) {
    thread.value.scrollTop += thread.value.scrollHeight - previousThreadHeight.value
    preservingHistoryScroll.value = false
    return
  }
  thread.value.scrollTo({ top: thread.value.scrollHeight })
}, { flush: 'post' })

async function loadMoreConversations() {
  if (!isOffline.value && conversationsQuery.hasNextPage.value && !conversationsQuery.isFetchingNextPage.value) {
    await conversationsQuery.fetchNextPage()
  }
}

async function loadOlderMessages() {
  if (!thread.value || isOffline.value || !messagesQuery.hasNextPage.value || messagesQuery.isFetchingNextPage.value) return
  previousThreadHeight.value = thread.value.scrollHeight
  preservingHistoryScroll.value = true
  await messagesQuery.fetchNextPage()
  if (messagesQuery.isFetchNextPageError.value) preservingHistoryScroll.value = false
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

function openConversation(name: string) {
  router.replace({ path: '/messages', query: { to: name } })
}

function closeConversation() {
  router.replace('/messages')
}

function submit() {
  const content = draft.value.trim()
  sendError.value = ''
  if (!forumStore.isAuthed) {
    forumStore.openAuth('/messages')
    return
  }
  if (isOffline.value) {
    sendError.value = '当前网络不可用，草稿已保留，恢复网络后再发送。'
    return
  }
  if (content && activePeer.value && !sendMutation.isPending.value) {
    sendMutation.mutate({ peer: activePeer.value, content })
  }
}

function formatTime(value: string) {
  return new Date(value).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <main class="detail-page messages-page" :class="{ 'has-active-thread': requestedPeer }">
    <button class="back-link messages-back" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section v-if="!forumStore.isAuthed" class="empty-state detail-empty-state public-page-state">
      <MessageCircle :size="34" />
      <h1>登录后查看私信</h1>
      <p>私信、会话和未读状态会跟随你的账号保存。登录后可以继续和同学、老师沟通。</p>
      <button class="primary-wide compact" type="button" @click="forumStore.openAuth('/messages')">登录 / 注册</button>
    </section>

    <section v-else class="messages-shell">
      <aside class="conversation-panel">
        <header>
          <div>
            <span>站内沟通</span>
            <h1>私信</h1>
          </div>
          <MessageCircle :size="24" />
        </header>
        <label class="conversation-search">
          <Search :size="17" />
          <input v-model="keyword" type="search" placeholder="搜索会话" />
        </label>
        <div v-if="conversationsQuery.isLoading.value" class="conversation-empty">
          <RefreshCcw class="state-spin" :size="28" />
          <strong>正在加载会话</strong>
          <p>请稍等，正在同步你的私信列表。</p>
        </div>
        <div v-else-if="isOffline" class="conversation-empty">
          <WifiOff :size="28" />
          <strong>当前网络不可用</strong>
          <p>恢复网络后可继续查看和发送私信。</p>
        </div>
        <div v-else-if="conversationsQuery.isError.value" class="conversation-empty">
          <MessageCircle :size="28" />
          <strong>会话加载失败</strong>
          <p>后端服务或网络暂时不可用，请稍后重试。</p>
          <button class="state-inline-button" type="button" @click="conversationsQuery.refetch()">重试</button>
        </div>
        <div v-else-if="filteredConversations.length" class="conversation-list">
          <button
            v-for="item in filteredConversations"
            :key="item.user.id"
            type="button"
            :class="{ active: item.user.nickname === activePeer }"
            @click="openConversation(item.user.nickname)"
          >
            <span class="conversation-avatar">{{ item.user.nickname.slice(0, 1) }}</span>
            <span class="conversation-copy">
              <strong>{{ item.user.nickname }}</strong>
              <small>{{ item.lastMessage }}</small>
            </span>
            <time>{{ formatTime(item.lastMessageAt) }}</time>
            <i v-if="item.unreadCount">{{ item.unreadCount }}</i>
          </button>
          <div v-if="conversationsQuery.hasNextPage.value || conversationsQuery.isFetchingNextPage.value || conversationsQuery.isFetchNextPageError.value" class="conversation-pagination">
            <p v-if="conversationsQuery.isFetchNextPageError.value" role="alert">更多会话加载失败，已加载会话仍可使用。</p>
            <button class="state-inline-button" type="button" :disabled="isOffline || conversationsQuery.isFetchingNextPage.value" @click="loadMoreConversations">
              {{ conversationsQuery.isFetchingNextPage.value ? '正在加载更多' : conversationsQuery.isFetchNextPageError.value ? '重试加载更多' : '加载更多会话' }}
            </button>
          </div>
        </div>
        <div v-else class="conversation-empty">
          <MessageCircle :size="28" />
          <strong>{{ conversationsEmptyTitle }}</strong>
          <p>{{ conversationsEmptyCopy }}</p>
        </div>
      </aside>

      <section v-if="activePeer" class="message-thread-panel">
        <header>
          <button class="thread-back" type="button" aria-label="返回会话列表" @click="closeConversation"><ChevronLeft :size="21" /></button>
          <span class="conversation-avatar">{{ activePeer.slice(0, 1) }}</span>
          <div>
            <strong>{{ activePeer }}</strong>
            <small>{{ activeConversation ? '站内用户' : '新会话' }}</small>
          </div>
        </header>
        <div ref="thread" class="message-thread" aria-live="polite">
          <div v-if="messagesQuery.isLoading.value" class="thread-empty">
            <RefreshCcw class="state-spin" :size="30" />
            <strong>正在加载消息</strong>
            <p>请稍等，正在打开与 {{ activePeer }} 的会话。</p>
          </div>
          <div v-else-if="messagesQuery.isError.value || isOffline" class="thread-empty">
            <WifiOff v-if="isOffline" :size="30" />
            <MessageCircle v-else :size="30" />
            <strong>{{ isOffline ? '网络已断开' : '消息加载失败' }}</strong>
            <p>{{ isOffline ? '恢复网络后再继续发送，避免内容丢失。' : '请重试，或返回会话列表重新打开。' }}</p>
            <button v-if="!isOffline" class="state-inline-button" type="button" @click="messagesQuery.refetch()">重试</button>
          </div>
          <div v-else-if="!messages.length" class="thread-empty">
            <MessageCircle :size="30" />
            <strong>开始一段真实对话</strong>
            <p>消息仅会发送给 {{ activePeer }}。</p>
          </div>
          <div v-if="messages.length && (messagesQuery.hasNextPage.value || messagesQuery.isFetchingNextPage.value || messagesQuery.isFetchNextPageError.value)" class="thread-empty">
            <p v-if="messagesQuery.isFetchNextPageError.value" role="alert">更早消息加载失败，当前消息不受影响。</p>
            <button class="state-inline-button" type="button" :disabled="isOffline || messagesQuery.isFetchingNextPage.value" @click="loadOlderMessages">
              {{ messagesQuery.isFetchingNextPage.value ? '正在加载更早消息' : messagesQuery.isFetchNextPageError.value ? '重试加载更早消息' : '加载更早消息' }}
            </button>
          </div>
          <article v-for="message in messages" :key="message.id" :class="{ mine: message.senderId === forumStore.currentUser?.id }">
            <p>{{ message.content }}</p>
            <time>{{ formatTime(message.createdAt) }}</time>
          </article>
        </div>
        <form class="message-composer" @submit.prevent="submit">
          <textarea v-model="draft" maxlength="2000" rows="2" :placeholder="`发消息给 ${activePeer}`" @keydown.ctrl.enter.prevent="submit" />
          <button type="submit" :disabled="isOffline || !draft.trim() || sendMutation.isPending.value" aria-label="发送消息"><Send :size="19" /></button>
        </form>
        <p v-if="isOffline" class="message-error">当前网络不可用，草稿会保留在输入框中，恢复网络后再发送。</p>
        <p v-if="sendError" class="message-error" role="alert">{{ sendError }}</p>
      </section>

      <section v-else class="message-thread-placeholder">
        <MessageCircle :size="38" />
        <h2>选择一个会话</h2>
        <p>也可以从用户主页点击“私信”发起新对话。</p>
      </section>
    </section>
  </main>
</template>
