<script setup lang="ts">
import { computed, ref, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Bookmark, ChevronLeft, MessageCircle, PenLine, Send, ShieldCheck, ThumbsUp } from '@lucide/vue'
import PostCard from '../components/PostCard.vue'
import { useAdviceEngagement } from '../composables/useAdviceEngagement'
import { adviceNotes } from '../lib/adviceNotes'
import { samplePosts } from '../lib/sampleData'
import { useForumStore } from '../stores/forum'

const route = useRoute()
const router = useRouter()
const forumStore = useForumStore()
const adviceEngagement = useAdviceEngagement()
const draft = ref('')
const commentInput = ref<HTMLInputElement | null>(null)
const noteId = computed(() => String(route.params.id ?? ''))
const note = computed(() => adviceNotes.find((item) => item.id === noteId.value))
const stats = computed(() => note.value ? adviceEngagement.noteStats(note.value.id, note.value.likes, note.value.saves).value : null)
const comments = computed(() => note.value ? adviceEngagement.noteComments(note.value.id).value : [])

const relatedPosts = computed(() => {
  if (!note.value) return []
  const tokens = [note.value.combination.replace(/\s|\+/g, ''), ...note.value.tags]
  return samplePosts
    .filter((post) =>
      tokens.some((token) =>
        post.title.includes(token) ||
        post.content.includes(token) ||
        post.tags.some((tag) => tag.includes(token) || token.includes(tag)),
      ),
    )
    .slice(0, 8)
})

const sections = computed(() => {
  if (!note.value) return []
  return [
    {
      title: '先看适配边界',
      body: note.value.summary,
    },
    {
      title: '怎么把建议落到纸面',
      body: `把“${note.value.combination}”拆成目标专业、成绩稳定性、学科压力和省份目录四张表。不要只听单一经验，先用官方目录排除硬限制，再看自己的学习证据。`,
    },
    {
      title: '适合继续追问的问题',
      body: `可以带着省份、最近三次成绩、目标专业和家庭偏好发帖追问。评论区越具体，认证老师和同组合同学越容易给出可执行建议。`,
    },
  ]
})

watchEffect(() => {
  if (route.hash === '#comments') {
    window.setTimeout(() => document.getElementById('comments')?.scrollIntoView({ behavior: 'smooth' }), 120)
  }
})

function toggleLike() {
  if (!note.value) return
  adviceEngagement.toggleLike(note.value.id, note.value.likes)
}

function toggleSave() {
  if (!note.value || !forumStore.requireAuth()) return
  adviceEngagement.toggleSave(note.value.id, note.value.saves)
}

function submitComment() {
  if (!note.value) return
  const content = draft.value.trim()
  if (!content) return
  if (!forumStore.requireAuth()) return
  adviceEngagement.addComment(note.value.id, content)
  draft.value = ''
}

function replyTo(author: string) {
  if (!forumStore.requireAuth()) return
  draft.value = `@${author} `
  window.setTimeout(() => commentInput.value?.focus(), 50)
}
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/advice')"><ChevronLeft :size="17" /> 返回建议库</button>

    <section v-if="note" class="advice-detail-layout">
      <article class="advice-detail-main">
        <div class="breadcrumb">首页 / 精选建议 / {{ note.lens }}</div>
        <div class="advice-note-visual advice-detail-visual" :class="`tone-${note.cover}`">
          <span>{{ note.metricLabel }}</span>
          <strong>{{ note.metric }}</strong>
          <small>{{ note.combination }}</small>
        </div>
        <small class="note-author-line">{{ note.author }} · {{ note.role }}</small>
        <h1>{{ note.title }}</h1>
        <p class="article-lead">{{ note.summary }}</p>

        <div class="mini-tag-row">
          <span v-for="tag in note.tags" :key="tag"># {{ tag }}</span>
        </div>

        <section v-for="section in sections" :key="section.title" class="advice-readable-section">
          <h2>{{ section.title }}</h2>
          <p>{{ section.body }}</p>
        </section>

        <section class="advice-action-board">
          <h2>行动清单</h2>
          <ol>
            <li v-for="action in note.actions" :key="action">{{ action }}</li>
          </ol>
        </section>

        <a class="note-source-link advice-source-large" :href="note.sourceUrl" target="_blank" rel="noreferrer">
          <ShieldCheck :size="16" /> {{ note.source }}
        </a>

        <div class="article-actions">
          <button :class="{ liked: stats?.liked }" @click="toggleLike">
            <ThumbsUp :size="17" /> {{ stats?.likes ?? note.likes }}
          </button>
          <button class="save-decision-button" :class="{ liked: stats?.saved }" @click="toggleSave">
            <Bookmark :size="17" /> {{ stats?.saved ? '已收藏' : '收藏' }}
          </button>
          <span><MessageCircle :size="17" /> {{ stats?.comments ?? 0 }} 评论</span>
        </div>
      </article>

      <aside class="article-side">
        <h2>继续提问</h2>
        <p>把省份、成绩波动、目标专业和担心点写清楚，建议会更接近真实决策。</p>
        <button class="primary-wide" @click="forumStore.openPublish('question')">
          <PenLine :size="16" /> 发布相关问题
        </button>
      </aside>
    </section>

    <section v-if="note" class="feed-panel topic-feed-panel">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">相关笔记</button>
        </div>
      </div>
      <div v-if="relatedPosts.length" class="feed-grid">
        <PostCard v-for="post in relatedPosts" :key="post.id" :post="post" />
      </div>
      <div v-else class="empty-state compact-empty">
        <h2>暂无直接关联帖子</h2>
        <p>可以发布一个相关问题，系统会沉淀到建议库。</p>
      </div>
    </section>

    <section v-if="note" id="comments" class="comment-board">
      <div class="comment-title-row">
        <h2>建议评论 {{ comments.length }}</h2>
        <span class="comment-sort-label">按时间</span>
      </div>
      <form class="comment-form detail-comment-form" @submit.prevent="submitComment">
        <input ref="commentInput" v-model="draft" :placeholder="forumStore.isAuthed ? '写下你的补充、追问或本省情况' : '登录后发表评论'" />
        <button :disabled="forumStore.isAuthed && !draft.trim()" type="submit">
          <Send :size="16" /> {{ forumStore.isAuthed ? '发表评论' : '登录评论' }}
        </button>
      </form>
      <div class="comment-list">
        <article v-for="comment in comments" :key="comment.id" class="comment-item">
          <RouterLink class="small-avatar user-link-avatar" :to="`/users/${encodeURIComponent(comment.author)}`">
            {{ comment.author.slice(0, 1) }}
          </RouterLink>
          <div>
            <div class="comment-meta">
              <RouterLink :to="`/users/${encodeURIComponent(comment.author)}`">
                <strong>{{ comment.author }}</strong>
              </RouterLink>
              <span>{{ comment.role }}</span>
            </div>
            <p>{{ comment.content }}</p>
            <div class="comment-actions">
              <span>{{ new Date(comment.createdAt).toLocaleString('zh-CN') }}</span>
              <button @click="replyTo(comment.author)">回复</button>
            </div>
          </div>
        </article>
      </div>
    </section>

    <section v-else class="empty-state">
      <h2>没有找到这条建议</h2>
      <p>请返回建议库重新选择。</p>
    </section>
  </main>
</template>
