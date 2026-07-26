<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { Bookmark, ChevronLeft, ChevronRight, ExternalLink, MessageSquare, RotateCcw, Send, ThumbsUp, UserPlus, X, ZoomIn, ZoomOut } from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiDataEnabled, createComment, fetchPostDetail, toggleFollowAuthor, togglePostFavorite, togglePostLike } from '../lib/api'
import { categoryLabels, roleLabels, subjectLabels, trackLabels } from '../lib/labels'
import { requirementData, sourcedDataPosts } from '../lib/realData'
import { appAssetUrl } from '../lib/runtime'
import { useForumStore } from '../stores/forum'

const route = useRoute()
const router = useRouter()
const queryClient = useQueryClient()
const forumStore = useForumStore()
const draft = ref('')
const commentError = ref('')
const commentInput = ref<HTMLInputElement | null>(null)
const activeImageIndex = ref<number | null>(null)
const zoom = ref(1)

const postId = computed(() => Number(route.params.id))
const detailQuery = useQuery({
  queryKey: computed(() => ['post-detail', postId.value, forumStore.session?.user.id ?? 'guest']),
  queryFn: () => fetchPostDetail(postId.value),
  enabled: apiDataEnabled,
})

const rawPost = computed(() => detailQuery.data.value?.post)
const post = computed(() => rawPost.value ? forumStore.hydratePost(rawPost.value) : undefined)
const comments = computed(() => detailQuery.data.value?.comments ?? [])
const displayedCommentCount = computed(() => comments.value.length)
const lightboxUrl = computed(() => activeImageIndex.value === null ? '' : post.value?.imageUrls?.[activeImageIndex.value] ?? '')
const dataEvidence = computed(() => {
  if (post.value?.category !== 'data') return null
  return sourcedDataPosts.find((item) => item.title === post.value?.title) ?? null
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
  onSuccess: (result) => {
    if (post.value) {
      forumStore.setUserFollow({
        name: post.value.authorName,
        role: post.value.authorRole,
        province: post.value.province,
        grade: post.value.grade,
        followedAt: new Date().toISOString(),
      }, result.active)
    }
    detailQuery.refetch()
  },
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

function toggleLike() {
  if (!post.value || !forumStore.requireAuth()) return
  likeMutation.mutate(undefined, { onError: () => { commentError.value = '点赞失败，请稍后重试。' } })
}

function toggleFavorite() {
  if (!post.value || !forumStore.requireAuth()) return
  favoriteMutation.mutate(undefined, { onError: () => { commentError.value = '收藏失败，请稍后重试。' } })
}

function toggleFollow() {
  if (!post.value || post.value.sourcePlatform || !forumStore.requireAuth()) return
  followMutation.mutate(undefined, { onError: () => { commentError.value = '关注失败，请稍后重试。' } })
}

function submitComment() {
  const content = draft.value.trim()
  if (!content) return
  if (!forumStore.isAuthed) {
    forumStore.authOpen = true
    return
  }
  commentError.value = ''
  commentMutation.mutate(content, {
    onError: () => {
      commentError.value = '评论发布失败，请确认已登录，且后端服务可用。'
    },
  })
}

function askCertifiedUser() {
  if (!forumStore.requireAuth()) return
  draft.value = '想请认证老师/规划师帮我看：'
  window.setTimeout(() => commentInput.value?.focus(), 50)
}

function replyTo(author: string) {
  if (!forumStore.requireAuth()) return
  draft.value = `@${author} `
  window.setTimeout(() => commentInput.value?.focus(), 50)
}

function openLightbox(index: number) {
  activeImageIndex.value = index
  zoom.value = 1
}

function closeLightbox() {
  activeImageIndex.value = null
  zoom.value = 1
}

function moveImage(offset: number) {
  const total = post.value?.imageUrls?.length ?? 0
  if (!total || activeImageIndex.value === null) return
  activeImageIndex.value = (activeImageIndex.value + offset + total) % total
  zoom.value = 1
}

function setZoom(nextZoom: number) {
  zoom.value = Math.min(4, Math.max(0.5, Number(nextZoom.toFixed(2))))
}

function handleImageWheel(event: WheelEvent) {
  setZoom(zoom.value + (event.deltaY < 0 ? 0.2 : -0.2))
}

function handleLightboxKeydown(event: KeyboardEvent) {
  if (activeImageIndex.value === null) return
  if (event.key === 'Escape') closeLightbox()
  else if (event.key === 'ArrowLeft') moveImage(-1)
  else if (event.key === 'ArrowRight') moveImage(1)
  else if (event.key === '+' || event.key === '=') setZoom(zoom.value + 0.2)
  else if (event.key === '-') setZoom(zoom.value - 0.2)
  else if (event.key === '0') setZoom(1)
}

watch(activeImageIndex, (index) => {
  document.body.style.overflow = index === null ? '' : 'hidden'
})
watch(postId, closeLightbox)
onMounted(() => window.addEventListener('keydown', handleLightboxKeydown))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleLightboxKeydown)
  document.body.style.overflow = ''
})
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>

    <section v-if="post" class="article-layout">
      <article class="article-main">
        <div class="breadcrumb">首页 / 帖子详情 / {{ categoryLabels[post.category] }}</div>
        <h1>{{ post.title }}</h1>
        <div class="article-meta">
          <a v-if="post.sourcePlatform" class="avatar medium source-detail-avatar" :href="post.sourceUrl" target="_blank" rel="noreferrer">
            <img v-if="post.sourceAvatarUrl" :src="appAssetUrl(post.sourceAvatarUrl)" :alt="post.sourceAuthor || post.authorName" />
            <template v-else>{{ (post.sourceAuthor || post.authorName).slice(0, 1) }}</template>
          </a>
          <RouterLink v-else class="avatar medium user-link-avatar" :to="`/users/${encodeURIComponent(post.authorName)}`">
            {{ post.authorName.slice(0, 1) }}
          </RouterLink>
          <span>
            <a v-if="post.sourcePlatform" class="author-name-link" :href="post.sourceUrl" target="_blank" rel="noreferrer">
              <strong>{{ post.sourceAuthor || post.authorName }} <em class="source-badge">小红书来源</em></strong>
            </a>
            <RouterLink v-else class="author-name-link" :to="`/users/${encodeURIComponent(post.authorName)}`">
              <strong>
                {{ post.authorName }}
                <em v-if="['teacher', 'counselor'].includes(post.authorRole)" class="verified-badge">认证</em>
              </strong>
            </RouterLink>
            <small v-if="post.sourcePlatform">原作者 · {{ post.province }} · 来源内容</small>
            <small v-else>{{ post.grade }} · {{ roleLabels[post.authorRole] }} · {{ post.province }}</small>
          </span>
          <a v-if="post.sourcePlatform" class="follow-button source-original-link" :href="post.sourceUrl" target="_blank" rel="noreferrer">
            <ExternalLink :size="15" /> 查看原文
          </a>
          <button v-else class="follow-button" :class="{ active: post.viewerFollowing }" @click="toggleFollow">
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
          <button class="article-gallery-main" type="button" aria-label="查看大图" @click="openLightbox(0)">
            <img :src="appAssetUrl(post.imageUrls[0])" :alt="post.title" />
            <span>查看大图 · {{ post.imageUrls.length }} 张</span>
          </button>
          <div v-if="post.imageUrls.length > 1" class="article-gallery-thumbs">
            <button v-for="(url, index) in post.imageUrls.slice(1)" :key="url" type="button" :aria-label="`查看第 ${index + 2} 张图片`" @click="openLightbox(index + 1)">
              <img :src="appAssetUrl(url)" :alt="`${post.title} 第 ${index + 2} 张图片`" />
            </button>
          </div>
        </div>
        <div v-else class="article-title-cover" :class="`cover-${post.track}`">
          <span>{{ trackLabels[post.track] }}</span>
          <strong>{{ post.title }}</strong>
          <small>{{ post.electives.map((item) => subjectLabels[item]).join(' · ') }}</small>
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
          <button :class="{ liked: post.viewerLiked }" @click="toggleLike">
            <ThumbsUp :size="17" /> {{ post.likesCount }}
          </button>
          <button class="save-decision-button" :class="{ liked: post.viewerFavorited }" @click="toggleFavorite">
            <Bookmark :size="17" /> {{ post.viewerFavorited ? '已收藏' : '收藏' }}
          </button>
          <span><MessageSquare :size="17" /> {{ displayedCommentCount }} 评论</span>
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
          <button type="button" @click="router.push('/settings')">先完善画像</button>
        </div>
      </aside>
    </section>

    <section v-else-if="!detailQuery.isLoading.value" class="empty-state detail-empty-state">
      <h1>帖子不存在或暂时无法加载</h1>
      <p>请返回论坛刷新列表，或稍后重试。</p>
      <button class="primary-wide compact" type="button" @click="router.push('/')">返回论坛</button>
    </section>

    <section v-if="post" class="comment-board">
      <div class="comment-title-row">
        <h2>全部评论 {{ comments.length }}</h2>
        <button @click="askCertifiedUser">向认证用户提问</button>
      </div>
      <div class="comment-guide">
        评论区会沉淀为后续搜索结果。补充省份、成绩稳定性、目标专业，认证老师/规划师更容易给出可执行建议。
      </div>
      <form class="comment-form detail-comment-form" @submit.prevent="submitComment">
        <input ref="commentInput" v-model="draft" :placeholder="forumStore.isAuthed ? '写下你的看法，帮助更多正在选科的人' : '登录后发表评论'" />
        <button :disabled="forumStore.isAuthed && (!draft.trim() || commentMutation.isPending.value)" type="submit">
          <Send :size="16" /> {{ forumStore.isAuthed ? '发表评论' : '登录评论' }}
        </button>
      </form>
      <p v-if="commentError" class="form-error">{{ commentError }}</p>
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
              <span>{{ roleLabels[comment.role] }}</span>
              <em v-if="['teacher', 'counselor'].includes(comment.role)" class="verified-badge">认证解答</em>
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
      <h2>正在加载帖子详情</h2>
      <p>如果长时间没有出现，请返回论坛重新选择帖子。</p>
    </section>

    <Teleport to="body">
      <div v-if="activeImageIndex !== null && lightboxUrl" class="image-lightbox" role="dialog" aria-modal="true" aria-label="帖子图片预览" @click.self="closeLightbox">
        <div class="lightbox-topbar">
          <span>{{ activeImageIndex + 1 }} / {{ post?.imageUrls?.length }}</span>
          <div class="lightbox-tools">
            <button type="button" aria-label="缩小图片" title="缩小" @click="setZoom(zoom - 0.2)"><ZoomOut :size="20" /></button>
            <strong>{{ Math.round(zoom * 100) }}%</strong>
            <button type="button" aria-label="放大图片" title="放大" @click="setZoom(zoom + 0.2)"><ZoomIn :size="20" /></button>
            <button type="button" aria-label="重置缩放" title="重置缩放" @click="setZoom(1)"><RotateCcw :size="19" /></button>
            <button type="button" aria-label="关闭图片预览" title="关闭" @click="closeLightbox"><X :size="22" /></button>
          </div>
        </div>
        <button v-if="(post?.imageUrls?.length ?? 0) > 1" class="lightbox-nav previous" type="button" aria-label="上一张" @click="moveImage(-1)"><ChevronLeft :size="30" /></button>
        <div class="lightbox-stage" @wheel.prevent="handleImageWheel">
          <img :src="appAssetUrl(lightboxUrl)" :alt="post?.title" :style="{ transform: `scale(${zoom})` }" />
        </div>
        <button v-if="(post?.imageUrls?.length ?? 0) > 1" class="lightbox-nav next" type="button" aria-label="下一张" @click="moveImage(1)"><ChevronRight :size="30" /></button>
      </div>
    </Teleport>
  </main>
</template>
