<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { Bookmark, ChevronLeft, MessageSquare, Send, ThumbsUp, UserPlus } from '@lucide/vue'
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { createComment, fetchPostDetail, toggleFollowAuthor, togglePostFavorite, togglePostLike } from '../lib/api'
import { categoryLabels, roleLabels, subjectLabels, trackLabels } from '../lib/labels'
import { requirementData, sourcedDataPosts } from '../lib/realData'
import { useForumStore } from '../stores/forum'

const route = useRoute()
const router = useRouter()
const queryClient = useQueryClient()
const forumStore = useForumStore()
const draft = ref('')
const commentError = ref('')

const postId = computed(() => Number(route.params.id))
const detailQuery = useQuery({
  queryKey: computed(() => ['post-detail', postId.value, forumStore.session?.user.id ?? 'guest']),
  queryFn: () => fetchPostDetail(postId.value),
})

const post = computed(() => detailQuery.data.value?.post)
const comments = computed(() => detailQuery.data.value?.comments ?? [])
const dataEvidence = computed(() => {
  if (post.value?.category !== 'data') return null
  return (
    sourcedDataPosts.find((item) => post.value?.title.includes(item.title.slice(0, 8))) ??
    sourcedDataPosts[0]
  )
})

const likeMutation = useMutation({
  mutationFn: () => togglePostLike(postId.value),
  onSuccess: () => detailQuery.refetch(),
})

const favoriteMutation = useMutation({
  mutationFn: () => togglePostFavorite(postId.value),
  onSuccess: () => detailQuery.refetch(),
})

const followMutation = useMutation({
  mutationFn: () => toggleFollowAuthor(post.value?.authorName ?? ''),
  onSuccess: () => detailQuery.refetch(),
})

const commentMutation = useMutation({
  mutationFn: (content: string) => createComment(postId.value, content),
  onSuccess: () => {
    draft.value = ''
    commentError.value = ''
    queryClient.invalidateQueries({ queryKey: ['post-detail', postId.value] })
    queryClient.invalidateQueries({ queryKey: ['posts'] })
    detailQuery.refetch()
  },
  onError: () => {
    commentError.value = '评论发布失败，请确认已登录，且内容不少于 2 个字。'
  },
})

function guarded(action: () => void) {
  if (forumStore.requireAuth()) action()
}

function submitComment() {
  const content = draft.value.trim()
  if (!content) return
  if (!forumStore.isAuthed) {
    forumStore.authOpen = true
    return
  }
  commentError.value = ''
  guarded(() => commentMutation.mutate(content))
}
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>

    <section v-if="post" class="article-layout">
      <article class="article-main">
        <div class="breadcrumb">首页 / 帖子详情 / {{ categoryLabels[post.category] }}</div>
        <h1>{{ post.title }}</h1>
        <div class="article-meta">
          <span class="avatar medium">{{ post.authorName.slice(0, 1) }}</span>
          <span>
            <strong>
              {{ post.authorName }}
              <em v-if="['teacher', 'counselor'].includes(post.authorRole)" class="verified-badge">认证</em>
            </strong>
            <small>{{ post.grade }} · {{ roleLabels[post.authorRole] }} · {{ post.province }}</small>
          </span>
          <button class="follow-button" :class="{ active: post.viewerFollowing }" @click="guarded(() => followMutation.mutate())">
            <UserPlus :size="15" /> {{ post.viewerFollowing ? '已关注' : '关注作者' }}
          </button>
        </div>
        <div class="article-tags">
          <span class="category-chip" :class="post.category">{{ categoryLabels[post.category] }}</span>
          <span>{{ trackLabels[post.track] }}</span>
          <span v-for="subject in post.electives" :key="subject">{{ subjectLabels[subject] }}</span>
          <span v-for="tag in post.tags" :key="tag"># {{ tag }}</span>
        </div>
        <p class="article-body">{{ post.content }}</p>

        <div v-if="post.imageUrls?.length" class="article-gallery">
          <img v-for="url in post.imageUrls" :key="url" :src="url" :alt="post.title" />
        </div>

        <section v-if="dataEvidence" class="post-data-evidence">
          <div>
            <small>真实数据来源</small>
            <h2>{{ dataEvidence.title }}</h2>
            <p>{{ dataEvidence.content }}</p>
            <a :href="dataEvidence.source.url" target="_blank" rel="noreferrer">
              {{ dataEvidence.source.publisher }}：{{ dataEvidence.source.label }}
            </a>
          </div>
          <div class="evidence-bars">
            <span
              v-for="slice in requirementData[0].slices"
              :key="slice.label"
              :style="{ '--bar-width': `${slice.value}%`, '--bar-color': slice.color }"
            >
              <strong>{{ slice.label }}</strong>
              <i></i>
              <em>{{ slice.value }}%</em>
            </span>
          </div>
        </section>

        <div class="article-actions">
          <button :class="{ liked: post.viewerLiked }" @click="guarded(() => likeMutation.mutate())">
            <ThumbsUp :size="17" /> {{ post.likesCount }}
          </button>
          <button class="save-decision-button" :class="{ liked: post.viewerFavorited }" @click="guarded(() => favoriteMutation.mutate())">
            <Bookmark :size="17" /> {{ post.viewerFavorited ? '已收藏' : '收藏' }}
          </button>
          <span><MessageSquare :size="17" /> {{ post.commentsCount }} 评论</span>
        </div>
      </article>

      <aside class="article-side">
        <h2>阅读建议</h2>
        <p>结合自身学科稳定性、目标专业和本省赋分规则判断，不要只看热门组合。</p>
        <button class="primary-wide" @click="forumStore.openPublish('question')">发布相关讨论</button>
        <div class="service-card">
          <small>服务入口预留</small>
          <strong>1v1 选科诊断</strong>
          <p>整理成绩、兴趣、目标专业和本省政策后，由认证规划师给出组合风险清单。</p>
          <button type="button">预约咨询</button>
        </div>
      </aside>
    </section>

    <section v-if="post" class="comment-board">
      <div class="comment-title-row">
        <h2>全部评论 {{ comments.length }}</h2>
        <button>向认证用户提问</button>
      </div>
      <div class="comment-guide">
        评论区会沉淀为后续搜索结果。补充省份、成绩稳定性、目标专业，认证老师/规划师更容易给出可执行建议。
      </div>
      <form class="comment-form detail-comment-form" @submit.prevent="submitComment">
        <input v-model="draft" :placeholder="forumStore.isAuthed ? '写下你的看法，帮助更多正在选科的人' : '登录后发表评论'" />
        <button :disabled="forumStore.isAuthed && (!draft.trim() || commentMutation.isPending.value)" type="submit">
          <Send :size="16" /> {{ forumStore.isAuthed ? '发表评论' : '登录评论' }}
        </button>
      </form>
      <p v-if="commentError" class="form-error">{{ commentError }}</p>
      <div class="comment-list">
        <article v-for="comment in comments" :key="comment.id" class="comment-item">
          <span class="small-avatar">{{ comment.author.slice(0, 1) }}</span>
          <div>
            <div class="comment-meta">
              <strong>{{ comment.author }}</strong>
              <span>{{ roleLabels[comment.role] }}</span>
              <em v-if="['teacher', 'counselor'].includes(comment.role)" class="verified-badge">认证解答</em>
            </div>
            <p>{{ comment.content }}</p>
            <div class="comment-actions">
              <span>{{ new Date(comment.createdAt).toLocaleString('zh-CN') }}</span>
              <button>回复</button>
            </div>
          </div>
        </article>
      </div>
    </section>

    <section v-else class="empty-state">
      <h2>正在加载帖子详情</h2>
      <p>如果长时间没有出现，请返回论坛重新选择帖子。</p>
    </section>
  </main>
</template>
