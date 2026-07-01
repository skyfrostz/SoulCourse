<script setup lang="ts">
import { ChevronLeft, Send, Star } from '@lucide/vue'
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useForumStore } from '../stores/forum'

const router = useRouter()
const forumStore = useForumStore()
const activeThread = ref(0)
const draft = ref('')
const threads = reactive([
  {
    name: '王老师',
    role: '高中选科指导',
    unread: 2,
    messages: [
      { from: 'them', text: '你现在不用急着定组合，先把近三次大考的物理、化学排名趋势拉出来。' },
      { from: 'them', text: '如果目标仍偏理工农医，建议重点核对物化双选要求，再看孩子能否承受。' },
    ],
  },
  {
    name: '选科研究所',
    role: '数据建议',
    unread: 1,
    messages: [
      { from: 'them', text: '我们已根据浙江、上海、甘肃公开目录补充了数据看板。' },
      { from: 'them', text: '你的个人画像还缺少 MBTI 和目标专业，完善后推荐会更具体。' },
    ],
  },
  {
    name: '海淀家长',
    role: '家长互助',
    unread: 0,
    messages: [{ from: 'them', text: '我们家最后是用“成绩稳定性 + 专业排除清单”定下来的，比单看覆盖率更安心。' }],
  },
])

function openThread(index: number) {
  forumStore.markMessagesRead(threads[index].unread)
  threads[index].unread = 0
  activeThread.value = index
}

onMounted(() => {
  forumStore.markMessagesRead()
  threads.forEach((thread) => {
    thread.unread = 0
  })
})

function sendMessage() {
  const text = draft.value.trim()
  if (!text) return
  threads[activeThread.value].messages.push({ from: 'me', text })
  draft.value = ''
}
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="messages-shell">
      <aside class="message-list">
        <div class="message-list-head">
          <h1>私信</h1>
          <span>{{ threads.reduce((sum, thread) => sum + thread.unread, 0) }} 条未读</span>
        </div>
        <button
          v-for="(thread, index) in threads"
          :key="thread.name"
          :class="{ active: activeThread === index }"
          type="button"
          @click="openThread(index)"
        >
          <span class="avatar">{{ thread.name.slice(0, 1) }}</span>
          <span>
            <strong>{{ thread.name }}</strong>
            <small>{{ thread.role }}</small>
          </span>
          <em v-if="thread.unread">{{ thread.unread }}</em>
        </button>
      </aside>

      <section class="message-thread">
        <header>
          <span class="avatar medium">{{ threads[activeThread].name.slice(0, 1) }}</span>
          <span>
            <strong>{{ threads[activeThread].name }}</strong>
            <small>{{ threads[activeThread].role }}</small>
          </span>
          <button type="button"><Star :size="16" /> 标记重点</button>
        </header>

        <div class="message-bubbles">
          <p v-for="message in threads[activeThread].messages" :key="message.text" :class="{ mine: message.from === 'me' }">
            {{ message.text }}
          </p>
        </div>

        <form class="message-compose" @submit.prevent="sendMessage">
          <input v-model="draft" placeholder="输入回复，和老师/家长继续沟通选科问题" />
          <button type="submit"><Send :size="16" /> 发送</button>
        </form>
      </section>
    </section>
  </main>
</template>
