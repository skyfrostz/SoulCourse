<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Bookmark, Brain, ChevronLeft, Mail, MessageSquare, PenLine, RefreshCcw, Settings, Sparkles, UserCheck, UserPlus, UserRound, Users, WifiOff } from '@lucide/vue'
import PostCard from '../components/PostCard.vue'
import { fetchProfile, toggleFollowAuthor } from '../lib/api'
import { roleLabels, subjectLabels, trackLabels } from '../lib/labels'
import { useForumStore } from '../stores/forum'
import type { Role } from '../types/forum'

const route = useRoute()
const router = useRouter()
const forumStore = useForumStore()
const queryClient = useQueryClient()
const activeProfileTab = ref<'posts' | 'comments' | 'favorites'>(route.hash === '#favorites' ? 'favorites' : 'posts')
const isOffline = ref(typeof navigator !== 'undefined' ? !navigator.onLine : false)

const profileName = computed(() => decodeURIComponent(String(route.params.name ?? '')))
const isCurrentUser = computed(() => forumStore.currentUser?.nickname === profileName.value)
const accountProfileQuery = useQuery({
  queryKey: ['profile', profileName.value],
  queryFn: () => fetchProfile(profileName.value),
  retry: false,
})
const accountProfile = computed(() => accountProfileQuery.data.value)
const allPosts = computed(() => accountProfile.value?.posts ?? [])
const authoredPosts = computed(() => accountProfile.value?.posts ?? [])
const authoredComments = computed(() => accountProfile.value?.comments.map((item) => item.comment) ?? [])

const profile = computed(() => {
  const fromPost = authoredPosts.value[0]
  const fromComment = authoredComments.value[0]
  if (accountProfile.value) {
    return {
      name: accountProfile.value.user.nickname,
      role: accountProfile.value.user.role,
      province: accountProfile.value.user.province,
      grade: accountProfile.value.user.grade,
    }
  }
  if (isCurrentUser.value && forumStore.currentUser) {
    return {
      name: forumStore.currentUser.nickname,
      role: forumStore.currentUser.role,
      province: forumStore.currentUser.province,
      grade: forumStore.currentUser.grade,
    }
  }
  return {
    name: profileName.value,
    role: fromPost?.authorRole ?? fromComment?.role ?? inferRole(profileName.value),
    province: fromPost?.province ?? '未公开',
    grade: fromPost?.grade ?? inferGrade(profileName.value),
  }
})

const favoritePosts = computed(() => isCurrentUser.value ? (accountProfile.value?.favorites ?? forumStore.getFavoritePosts(allPosts.value)) : [])
const followingList = computed(() => accountProfile.value?.following ?? [])
const followerList = computed(() => accountProfile.value?.followers ?? [])
const isFollowing = computed(() => accountProfile.value?.viewerFollowing ?? false)
const followMutation = useMutation({
  mutationFn: () => toggleFollowAuthor(profile.value.name),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['profile'] })
    queryClient.invalidateQueries({ queryKey: ['posts'] })
  },
})
const commentCards = computed(() =>
  accountProfile.value?.comments.map((item) => ({ comment: item.comment, postId: item.comment.postId, postTitle: item.postTitle })) ?? [],
)
const profileStats = computed(() => accountProfile.value?.stats ?? {
  posts: authoredPosts.value.length,
  comments: authoredComments.value.length,
  following: followingList.value.length,
  followers: followerList.value.length,
  favorites: favoritePosts.value.length,
  engagement: 0,
})
const choiceProfile = computed(() => accountProfile.value?.choiceProfile ?? forumStore.choiceProfile)
const profileCompletion = computed(() => {
  const values = [choiceProfile.value.mbti, choiceProfile.value.targetMajors, choiceProfile.value.gradeRank, choiceProfile.value.city]
  return Math.round((values.filter(Boolean).length / values.length) * 100)
})

function roleTone(role: Role) {
  if (role === 'teacher' || role === 'counselor') return '认证用户'
  if (role === 'parent') return '家长视角'
  return '学生经验'
}

function inferRole(name: string): Role {
  if (name.includes('老师')) return 'teacher'
  if (name.includes('规划师')) return 'counselor'
  if (name.includes('家长')) return 'parent'
  return 'student'
}

function inferGrade(name: string) {
  if (name.includes('老师')) return '教师'
  if (name.includes('规划师')) return '规划师'
  if (name.includes('家长')) return '家长'
  return '选科用户'
}

function toggleFollow() {
  if (forumStore.requireAuth()) followMutation.mutate()
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
  <main class="detail-page user-profile-page">
    <button class="back-link" @click="router.back()"><ChevronLeft :size="17" /> 返回上一页</button>

    <section v-if="accountProfileQuery.isLoading.value" class="empty-state public-page-state">
      <RefreshCcw class="state-spin" :size="34" />
      <h1>正在加载用户主页</h1>
      <p>请稍等，正在同步公开帖子、评论和关注信息。</p>
    </section>

    <section v-else-if="isOffline || accountProfileQuery.isError.value || !accountProfile" class="empty-state detail-empty-state public-page-state">
      <WifiOff v-if="isOffline" :size="34" />
      <UserRound v-else :size="34" />
      <h1>{{ isOffline ? '当前网络不可用' : '用户主页加载失败' }}</h1>
      <p>{{ isOffline ? '恢复网络后可以重新加载这个用户主页。' : '这个用户可能不存在，或页面数据暂时无法同步。' }}</p>
      <div class="state-action-row">
        <button v-if="!isOffline" class="primary-wide compact" type="button" @click="accountProfileQuery.refetch()">重试</button>
        <RouterLink class="ghost-button compact" to="/">返回首页</RouterLink>
      </div>
    </section>

    <template v-else>
    <section class="user-profile-hero">
      <div class="user-profile-avatar">{{ profile.name.slice(0, 1) }}</div>
      <div>
        <div class="breadcrumb">用户主页 / {{ roleTone(profile.role) }}</div>
        <h1>{{ profile.name }}</h1>
        <p>{{ profile.grade }} · {{ roleLabels[profile.role] }} · {{ profile.province }}</p>
        <p v-if="accountProfile?.bio" class="profile-bio">{{ accountProfile.bio }}</p>
        <div class="overview-metrics">
          <span><PenLine :size="17" /> {{ profileStats.posts }} 篇帖子</span>
          <span><MessageSquare :size="17" /> {{ profileStats.comments }} 条评论</span>
          <span><UserPlus :size="17" /> {{ profileStats.following }} 关注</span>
          <span><Users :size="17" /> {{ profileStats.followers }} 粉丝</span>
          <span v-if="isCurrentUser"><Bookmark :size="17" /> {{ profileStats.favorites }} 个收藏</span>
        </div>
      </div>
      <div class="user-profile-actions">
        <RouterLink v-if="isCurrentUser" class="write-button" to="/settings">
          <Settings :size="16" /> 编辑资料
        </RouterLink>
        <RouterLink v-else-if="accountProfile" class="write-button" :to="`/messages?to=${encodeURIComponent(profile.name)}`">
          <Mail :size="16" /> 私信
        </RouterLink>
        <button v-if="!isCurrentUser" class="follow-button" :class="{ active: isFollowing }" type="button" :disabled="followMutation.isPending.value" @click="toggleFollow">
          <component :is="isFollowing ? UserCheck : UserPlus" :size="16" />
          {{ isFollowing ? '已关注' : '关注' }}
        </button>
      </div>
    </section>

    <section v-if="isCurrentUser" class="profile-choice-card">
      <div>
        <small><Sparkles :size="15" /> 我的选科画像</small>
        <h2>{{ trackLabels[choiceProfile.preferredTrack] }} · {{ choiceProfile.preferredSubjects.map((item) => subjectLabels[item]).join(' + ') }}</h2>
        <p>MBTI：{{ choiceProfile.mbti || '未填写' }} · 目标专业：{{ choiceProfile.targetMajors || '未填写' }}</p>
      </div>
      <RouterLink to="/settings">画像完成度 {{ profileCompletion }}%</RouterLink>
    </section>

    <nav v-if="isCurrentUser" class="profile-service-grid mobile-profile-shortcuts" aria-label="个人功能">
      <RouterLink to="/settings">
        <span class="tone-profile"><Sparkles :size="21" /></span>
        <strong>选科画像</strong>
        <small>{{ profileCompletion }}% 完成</small>
      </RouterLink>
      <RouterLink to="/settings">
        <span class="tone-mbti"><Brain :size="21" /></span>
        <strong>MBTI 偏好</strong>
        <small>{{ choiceProfile.mbti || '待填写' }}</small>
      </RouterLink>
      <RouterLink to="/following">
        <span class="tone-following"><Users :size="21" /></span>
        <strong>我的关注</strong>
        <small>{{ profileStats.following }} 人</small>
      </RouterLink>
      <button type="button" @click="activeProfileTab = 'favorites'">
        <span class="tone-favorite"><Bookmark :size="21" /></span>
        <strong>我的收藏</strong>
        <small>{{ profileStats.favorites }} 篇</small>
      </button>
    </nav>

    <nav class="mobile-profile-tabs" aria-label="个人主页内容">
      <button type="button" :class="{ active: activeProfileTab === 'posts' }" @click="activeProfileTab = 'posts'">帖子</button>
      <button type="button" :class="{ active: activeProfileTab === 'comments' }" @click="activeProfileTab = 'comments'">评论</button>
      <button v-if="isCurrentUser" type="button" :class="{ active: activeProfileTab === 'favorites' }" @click="activeProfileTab = 'favorites'">收藏</button>
    </nav>

    <section class="mobile-profile-content">
      <template v-if="activeProfileTab === 'posts'">
        <div v-if="authoredPosts.length" class="feed-grid">
          <PostCard v-for="post in authoredPosts" :key="post.id" :post="post" />
        </div>
        <div v-else class="empty-state compact-empty">
          <UserRound :size="28" />
          <h2>还没有公开帖子</h2>
          <p>发布后的选科经验会显示在这里。</p>
        </div>
      </template>
      <template v-else-if="activeProfileTab === 'comments'">
        <div v-if="commentCards.length" class="profile-comment-list">
          <RouterLink v-for="item in commentCards" :key="item.comment.id" :to="`/posts/${item.postId}`">
            <span>{{ item.postTitle }}</span>
            <p>{{ item.comment.content }}</p>
            <small>{{ new Date(item.comment.createdAt).toLocaleString('zh-CN') }}</small>
          </RouterLink>
        </div>
        <div v-else class="empty-state compact-empty">
          <MessageSquare :size="28" />
          <h2>还没有评论</h2>
          <p>参与过的讨论会显示在这里。</p>
        </div>
      </template>
      <template v-else>
        <div v-if="favoritePosts.length" class="feed-grid">
          <PostCard v-for="post in favoritePosts" :key="post.id" :post="post" />
        </div>
        <div v-else class="empty-state compact-empty">
          <Bookmark :size="28" />
          <h2>还没有收藏</h2>
          <p>收藏重要经验，建立自己的选科资料夹。</p>
        </div>
      </template>
    </section>

    <section v-if="isCurrentUser" class="profile-section follow-management-section desktop-profile-content">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">我的关注</button>
        </div>
        <span class="profile-section-count">{{ profileStats.following }} 人</span>
      </div>
      <div v-if="followingList.length" class="follow-card-grid">
        <RouterLink
          v-for="user in followingList"
          :key="user.name"
          class="follow-user-card"
          :to="`/users/${encodeURIComponent(user.name)}`"
        >
          <span class="small-avatar">{{ user.name.slice(0, 1) }}</span>
          <span>
            <strong>{{ user.name }}</strong>
            <small>{{ user.grade }} · {{ roleLabels[user.role] }} · {{ user.province }}</small>
          </span>
          <em>查看主页</em>
        </RouterLink>
      </div>
      <div v-else class="empty-state compact-empty">
        <UserPlus :size="28" />
        <h2>还没有关注用户</h2>
        <p>在帖子作者页或评论用户页点击关注，会集中出现在这里。</p>
      </div>
    </section>

    <section v-if="isCurrentUser" class="profile-section follow-management-section desktop-profile-content">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">关注我的</button>
        </div>
        <span class="profile-section-count">{{ profileStats.followers }} 人</span>
      </div>
      <div v-if="followerList.length" class="follow-card-grid">
        <RouterLink
          v-for="user in followerList"
          :key="user.name"
          class="follow-user-card"
          :to="`/users/${encodeURIComponent(user.name)}`"
        >
          <span class="small-avatar">{{ user.name.slice(0, 1) }}</span>
          <span>
            <strong>{{ user.name }}</strong>
            <small>{{ user.grade }} · {{ roleLabels[user.role] }} · {{ user.province }}</small>
          </span>
          <em>查看主页</em>
        </RouterLink>
      </div>
      <div v-else class="empty-state compact-empty">
        <Users :size="28" />
        <h2>还没有人关注你</h2>
        <p>多发经验帖、在评论区认真回答问题，会让更多同学关注你。</p>
      </div>
    </section>

    <section class="profile-section desktop-profile-content">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">TA 的公开笔记</button>
        </div>
      </div>
      <div v-if="authoredPosts.length" class="feed-grid">
        <PostCard v-for="post in authoredPosts" :key="post.id" :post="post" />
      </div>
      <div v-else class="empty-state compact-empty">
        <UserRound :size="28" />
        <h2>还没有公开帖子</h2>
        <p>可以先查看 TA 在评论区留下的选科讨论。</p>
      </div>
    </section>

    <section class="profile-section desktop-profile-content">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">TA 的评论</button>
        </div>
      </div>
      <div v-if="commentCards.length" class="profile-comment-list">
        <RouterLink v-for="item in commentCards" :key="item.comment.id" :to="`/posts/${item.postId}`">
          <span>{{ item.postTitle }}</span>
          <p>{{ item.comment.content }}</p>
          <small>{{ new Date(item.comment.createdAt).toLocaleString('zh-CN') }}</small>
        </RouterLink>
      </div>
      <div v-else class="empty-state compact-empty">
        <MessageSquare :size="28" />
        <h2>还没有评论</h2>
        <p>评论区的提问和补充会沉淀在这里。</p>
      </div>
    </section>

    <section v-if="isCurrentUser" class="profile-section desktop-profile-content">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">我的收藏</button>
        </div>
      </div>
      <div v-if="favoritePosts.length" class="feed-grid">
        <PostCard v-for="post in favoritePosts" :key="post.id" :post="post" />
      </div>
      <div v-else class="empty-state compact-empty">
        <Bookmark :size="28" />
        <h2>还没有收藏</h2>
        <p>把重要经验帖先收藏，后续可以作为自己的选科档案。</p>
      </div>
    </section>
    </template>
  </main>
</template>
