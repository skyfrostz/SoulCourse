<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { Bookmark, ChevronLeft, ChevronRight, ExternalLink, Flag, MessageSquare, Pencil, RotateCcw, Save, Send, ThumbsUp, Trash2, UserPlus, WifiOff, X, ZoomIn, ZoomOut } from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { apiDataEnabled, createComment, deletePost, fetchPostDetail, reportPost, toggleFollowAuthor, togglePostFavorite, togglePostLike, updatePost } from '../lib/api'
import { categoryLabels, roleLabels, subjectLabels, trackLabels } from '../lib/labels'
import { appAssetUrl } from '../lib/runtime'
import { useForumStore } from '../stores/forum'
import type { Category, Subject, Track } from '../types/forum'

const route = useRoute()
const router = useRouter()
const queryClient = useQueryClient()
const forumStore = useForumStore()
const draft = ref('')
const commentError = ref('')
const reportMessage = ref('')
const postActionMessage = ref('')
const editError = ref('')
const editing = ref(false)
const commentInput = ref<HTMLInputElement | null>(null)
const activeImageIndex = ref<number | null>(null)
const zoom = ref(1)
const isOffline = ref(typeof navigator !== 'undefined' ? !navigator.onLine : false)

const postId = computed(() => Number(route.params.id))
const postDetailQueryKey = computed(() => ['post-detail', postId.value, forumStore.session?.user.id ?? 'guest'] as const)
const detailQuery = useQuery({
  queryKey: postDetailQueryKey,
  queryFn: () => fetchPostDetail(postId.value),
  enabled: apiDataEnabled,
})

const rawPost = computed(() => detailQuery.data.value?.post)
const postPatch = ref<Partial<NonNullable<typeof rawPost.value>>>({})
const post = computed(() => rawPost.value ? forumStore.hydratePost({ ...rawPost.value, ...postPatch.value }) : undefined)
const comments = computed(() => detailQuery.data.value?.comments ?? [])
const displayedCommentCount = computed(() => comments.value.length)
const lightboxUrl = computed(() => activeImageIndex.value === null ? '' : post.value?.imageUrls?.[activeImageIndex.value] ?? '')
const dataEvidence = computed(() => {
  if (post.value?.category !== 'data' || !post.value.sourceUrl) return null
  return {
    title: post.value.sourceTitle || post.value.title,
    content: post.value.content,
    publisher: post.value.sourcePlatform || '官方来源',
    url: post.value.sourceUrl,
  }
})
const viewerOwnsPost = computed(() => Boolean(post.value?.userId && forumStore.currentUser?.id === post.value.userId))
const subjectOptions: Subject[] = ['chemistry', 'biology', 'politics', 'geography']
const trackOptions: Track[] = ['physics', 'history']
const categoryOptions: Category[] = ['question', 'experience', 'data']
const editForm = reactive<{
  title: string
  content: string
  tags: string
  track: Track
  electives: Subject[]
  category: Category
}>({
  title: '',
  content: '',
  tags: '',
  track: 'physics',
  electives: ['chemistry', 'biology'],
  category: 'question',
})

const likeMutation = useMutation({
  mutationFn: () => togglePostLike(postId.value),
  onSuccess: (result) => {
    postPatch.value = { ...postPatch.value, viewerLiked: result.active, likesCount: result.count }
    detailQuery.refetch()
  },
})

const favoriteMutation = useMutation({
  mutationFn: () => togglePostFavorite(postId.value),
  onSuccess: (result) => {
    postPatch.value = { ...postPatch.value, viewerFavorited: result.active, favoritesCount: result.count }
    detailQuery.refetch()
  },
})

const followMutation = useMutation({
  mutationFn: () => toggleFollowAuthor(post.value?.authorName ?? ''),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['profile'] })
    detailQuery.refetch()
  },
})

const commentMutation = useMutation({
  mutationFn: (content: string) => createComment(postId.value, content),
  onSuccess: (created) => {
    draft.value = ''
    commentError.value = ''
    queryClient.setQueryData<{ post: NonNullable<typeof rawPost.value>; comments: typeof comments.value }>(
      postDetailQueryKey.value,
      (current) => current && !current.comments.some((comment) => comment.id === created.id)
        ? { ...current, comments: [...current.comments, created] }
        : current,
    )
    queryClient.invalidateQueries({ queryKey: ['posts'] })
  },
  onError: () => {
    commentError.value = '评论发布失败，请确认已登录，且内容不少于 2 个字。'
  },
})

const reportMutation = useMutation({
  mutationFn: (reason: string) => reportPost(postId.value, reason),
  onSuccess: () => {
    reportMessage.value = '举报已提交，管理员会尽快审核。'
  },
  onError: () => {
    reportMessage.value = '举报提交失败，请稍后重试。'
  },
})

const deleteMutation = useMutation({
  mutationFn: () => deletePost(postId.value),
  onSuccess: async () => {
    postActionMessage.value = '帖子已删除，正在返回论坛。'
    await queryClient.invalidateQueries({ queryKey: ['posts'] })
    await router.push('/')
  },
  onError: () => {
    postActionMessage.value = '删除失败，请确认这是你发布的帖子，或稍后重试。'
  },
})

const updateMutation = useMutation({
  mutationFn: () => updatePost(postId.value, {
    title: editForm.title.trim(),
    content: editForm.content.trim(),
    tags: editForm.tags.split(/[，,\s#]+/).map((item) => item.trim()).filter(Boolean).slice(0, 8),
    track: editForm.track,
    electives: editForm.electives,
    category: editForm.category,
  }),
  onSuccess: (updated) => {
    editError.value = ''
    postActionMessage.value = '帖子已更新。'
    postPatch.value = { ...postPatch.value, ...updated }
    queryClient.invalidateQueries({ queryKey: ['posts'] })
    queryClient.invalidateQueries({ queryKey: ['post-detail', postId.value] })
    editing.value = false
  },
  onError: () => {
    editError.value = '保存失败，请确认标题、正文和两个再选科目填写完整。'
  },
})

function syncEditForm() {
  if (!post.value) return
  editForm.title = post.value.title
  editForm.content = post.value.content
  editForm.tags = post.value.tags.join(' ')
  editForm.track = post.value.track as Track
  editForm.electives = [...post.value.electives] as Subject[]
  editForm.category = post.value.category as Category
}

function toggleLike() {
  if (likeMutation.isPending.value) return
  if (isOffline.value) {
    commentError.value = '当前网络不可用，恢复连接后再点赞。'
    return
  }
  if (!post.value || !forumStore.requireAuth()) return
  likeMutation.mutate(undefined, { onError: () => { commentError.value = '点赞失败，请稍后重试。' } })
}

function toggleFavorite() {
  if (favoriteMutation.isPending.value) return
  if (isOffline.value) {
    commentError.value = '当前网络不可用，恢复连接后再收藏。'
    return
  }
  if (!post.value || !forumStore.requireAuth()) return
  favoriteMutation.mutate(undefined, { onError: () => { commentError.value = '收藏失败，请稍后重试。' } })
}

function toggleFollow() {
  if (followMutation.isPending.value) return
  if (isOffline.value) {
    commentError.value = '当前网络不可用，恢复连接后再关注。'
    return
  }
  if (!post.value || post.value.sourcePlatform || !forumStore.requireAuth()) return
  followMutation.mutate(undefined, { onError: () => { commentError.value = '关注失败，请稍后重试。' } })
}

function submitComment() {
  if (commentMutation.isPending.value) return
  if (isOffline.value) {
    commentError.value = '当前网络不可用，恢复连接后再发表评论。'
    return
  }
  if (!forumStore.isAuthed) {
    forumStore.authOpen = true
    return
  }
  const content = draft.value.trim()
  if (!content) {
    commentInput.value?.focus()
    return
  }
  commentError.value = ''
  commentMutation.mutate(content, {
    onError: () => {
      commentError.value = '评论发布失败，请确认已登录，且后端服务可用。'
    },
  })
}

function submitReport() {
  if (isOffline.value) {
    reportMessage.value = '当前网络不可用，恢复连接后再提交举报。'
    return
  }
  if (!post.value || !forumStore.requireAuth()) return
  const reason = window.prompt('请简要说明举报原因')
  if (!reason?.trim()) return
  reportMessage.value = ''
  reportMutation.mutate(reason.trim())
}

function deleteOwnPost() {
  if (!post.value || deleteMutation.isPending.value) return
  if (isOffline.value) {
    postActionMessage.value = '当前网络不可用，恢复连接后再删除帖子。'
    return
  }
  if (!forumStore.requireAuth()) return
  if (!viewerOwnsPost.value) {
    postActionMessage.value = '只能删除你自己发布的帖子。'
    return
  }
  if (!window.confirm(`确认删除「${post.value.title}」？删除后其他用户将无法继续查看。`)) return
  postActionMessage.value = ''
  deleteMutation.mutate()
}

function startEditingPost() {
  if (!viewerOwnsPost.value || !post.value) return
  syncEditForm()
  editError.value = ''
  postActionMessage.value = ''
  editing.value = true
}

function cancelEditingPost() {
  if (updateMutation.isPending.value) return
  editing.value = false
  editError.value = ''
}

function toggleEditSubject(subject: Subject) {
  if (editForm.electives.includes(subject)) {
    if (editForm.electives.length > 1) {
      editForm.electives = editForm.electives.filter((item) => item !== subject)
    }
    return
  }
  editForm.electives = [...editForm.electives.slice(-1), subject]
}

function saveEditedPost() {
  if (updateMutation.isPending.value) return
  if (isOffline.value) {
    editError.value = '当前网络不可用，恢复连接后再保存。'
    return
  }
  if (!editForm.title.trim() || !editForm.content.trim() || editForm.electives.length !== 2) {
    editError.value = '请填写标题、正文，并选择两个再选科目。'
    return
  }
  editError.value = ''
  updateMutation.mutate()
}

function focusComments() {
  document.getElementById('post-comments')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  window.setTimeout(() => commentInput.value?.focus(), 250)
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

function updateOnlineState() {
  isOffline.value = typeof navigator !== 'undefined' ? !navigator.onLine : false
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
watch(post, () => {
  if (!editing.value) syncEditForm()
})
watch(postId, closeLightbox)
onMounted(() => {
  window.addEventListener('keydown', handleLightboxKeydown)
  window.addEventListener('online', updateOnlineState)
  window.addEventListener('offline', updateOnlineState)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleLightboxKeydown)
  window.removeEventListener('online', updateOnlineState)
  window.removeEventListener('offline', updateOnlineState)
  document.body.style.overflow = ''
})
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>

    <section v-if="post" class="article-layout">
      <article class="article-main">
        <div class="breadcrumb">首页 / 帖子详情 / {{ categoryLabels[post.category] }}</div>
        <h1 v-if="!editing">{{ post.title }}</h1>
        <form v-else class="post-edit-panel" @submit.prevent="saveEditedPost">
          <label>
            <span>标题</span>
            <input v-model="editForm.title" maxlength="80" required />
          </label>
          <label>
            <span>正文</span>
            <textarea v-model="editForm.content" maxlength="4000" rows="6" required />
          </label>
          <div class="post-edit-options">
            <label>
              <span>类型</span>
              <select v-model="editForm.category">
                <option v-for="item in categoryOptions" :key="item" :value="item">{{ categoryLabels[item] }}</option>
              </select>
            </label>
            <label>
              <span>方向</span>
              <select v-model="editForm.track">
                <option v-for="item in trackOptions" :key="item" :value="item">{{ trackLabels[item] }}</option>
              </select>
            </label>
          </div>
          <div class="post-edit-subjects" aria-label="再选科目">
            <button
              v-for="subject in subjectOptions"
              :key="subject"
              type="button"
              :class="{ active: editForm.electives.includes(subject) }"
              @click="toggleEditSubject(subject)"
            >
              {{ subjectLabels[subject] }}
            </button>
          </div>
          <label>
            <span>标签</span>
            <input v-model="editForm.tags" maxlength="160" placeholder="用空格或逗号分隔，最多 8 个" />
          </label>
          <p v-if="editError" class="form-error">{{ editError }}</p>
          <div class="post-edit-actions">
            <button type="button" :disabled="updateMutation.isPending.value" @click="cancelEditingPost">取消</button>
            <button class="primary" type="submit" :disabled="updateMutation.isPending.value">
              <Save :size="16" /> {{ updateMutation.isPending.value ? '保存中...' : '保存修改' }}
            </button>
          </div>
        </form>
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
          <button v-else class="follow-button" :class="{ active: post.viewerFollowing }" :disabled="isOffline || followMutation.isPending.value" @click="toggleFollow">
            <UserPlus :size="15" /> {{ followMutation.isPending.value ? '处理中...' : post.viewerFollowing ? '已关注' : '关注作者' }}
          </button>
        </div>
        <div v-if="!editing" class="article-tags">
          <span class="category-chip" :class="post.category">{{ categoryLabels[post.category] }}</span>
          <span>{{ trackLabels[post.track] }}</span>
          <span v-for="subject in post.electives" :key="subject">{{ subjectLabels[subject] }}</span>
          <span v-for="tag in post.tags" :key="tag"># {{ tag }}</span>
        </div>
        <p v-if="!editing" class="article-body">{{ post.content }}</p>

        <div v-if="!editing && post.imageUrls?.length" class="article-gallery">
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
        <div v-else-if="!editing" class="article-title-cover" :class="`cover-${post.track}`">
          <span>{{ trackLabels[post.track] }}</span>
          <strong>{{ post.title }}</strong>
          <small>{{ post.electives.map((item) => subjectLabels[item]).join(' · ') }}</small>
        </div>

        <section v-if="dataEvidence" class="post-data-evidence">
          <div>
            <small>真实数据来源</small>
            <h2>{{ dataEvidence.title }}</h2>
            <p>{{ dataEvidence.content }}</p>
            <a :href="dataEvidence.url" target="_blank" rel="noreferrer">
              {{ dataEvidence.publisher }}：查看原始来源
            </a>
          </div>
        </section>

        <div class="article-actions">
          <button :class="{ liked: post.viewerLiked }" :disabled="isOffline || likeMutation.isPending.value" @click="toggleLike">
            <ThumbsUp :size="17" /> {{ post.likesCount }}
          </button>
          <button class="save-decision-button" :class="{ liked: post.viewerFavorited }" :disabled="isOffline || favoriteMutation.isPending.value" @click="toggleFavorite">
            <Bookmark :size="17" /> {{ favoriteMutation.isPending.value ? '处理中...' : post.viewerFavorited ? '已收藏' : '收藏' }}
          </button>
          <button type="button" @click="focusComments"><MessageSquare :size="17" /> {{ displayedCommentCount }} 评论</button>
          <button type="button" :disabled="isOffline || reportMutation.isPending.value" @click="submitReport"><Flag :size="17" /> 举报</button>
          <button v-if="viewerOwnsPost" type="button" :disabled="isOffline || updateMutation.isPending.value" @click="startEditingPost">
            <Pencil :size="17" /> 编辑帖子
          </button>
          <button v-if="viewerOwnsPost" class="danger-inline-action" type="button" :disabled="isOffline || deleteMutation.isPending.value" @click="deleteOwnPost">
            <Trash2 :size="17" /> {{ deleteMutation.isPending.value ? '删除中...' : '删除帖子' }}
          </button>
        </div>
        <p v-if="reportMessage" class="form-success">{{ reportMessage }}</p>
        <p v-if="postActionMessage" :class="deleteMutation.isError.value ? 'form-error' : 'form-success'">{{ postActionMessage }}</p>
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

    <section v-else-if="!detailQuery.isLoading.value" class="empty-state detail-empty-state public-page-state">
      <WifiOff v-if="isOffline" :size="34" />
      <MessageSquare v-else :size="34" />
      <h1>{{ isOffline ? '当前网络不可用' : detailQuery.isError.value ? '帖子加载失败' : '帖子不存在或暂时无法加载' }}</h1>
      <p>{{ isOffline ? '恢复网络后可重新加载帖子详情。' : detailQuery.isError.value ? '后端服务暂时不可用，请稍后重试。' : '请返回论坛刷新列表，或稍后重试。' }}</p>
      <div class="state-action-row">
        <button v-if="!isOffline && detailQuery.isError.value" class="primary-wide compact" type="button" @click="detailQuery.refetch()">重试</button>
        <button class="primary-wide compact" type="button" @click="router.push('/')">返回论坛</button>
      </div>
    </section>

    <section v-if="post" id="post-comments" class="comment-board">
      <div class="comment-title-row">
        <h2>全部评论 {{ comments.length }}</h2>
        <button @click="askCertifiedUser">向认证用户提问</button>
      </div>
      <div class="comment-guide">
        评论区会沉淀为后续搜索结果。补充省份、成绩稳定性、目标专业，认证老师/规划师更容易给出可执行建议。
      </div>
      <form class="comment-form detail-comment-form" @submit.prevent="submitComment">
        <input ref="commentInput" v-model="draft" :placeholder="forumStore.isAuthed ? '写下你的看法，帮助更多正在选科的人' : '登录后发表评论'" />
        <button :disabled="isOffline || (forumStore.isAuthed && (!draft.trim() || commentMutation.isPending.value))" type="submit">
          <Send :size="16" /> {{ commentMutation.isPending.value ? '发布中...' : forumStore.isAuthed ? '发表评论' : '登录评论' }}
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
        <div v-if="!comments.length" class="empty-state compact-empty comment-empty-state">
          <MessageSquare :size="28" />
          <h2>还没有评论</h2>
          <p>{{ forumStore.isAuthed ? '你可以成为第一个补充经验的人。' : '登录后可以参与讨论。' }}</p>
        </div>
      </div>
    </section>

    <section v-else-if="detailQuery.isLoading.value" class="empty-state public-page-state">
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
