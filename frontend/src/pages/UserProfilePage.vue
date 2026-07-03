<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Bookmark, ChevronLeft, MessageSquare, PenLine, Settings, Sparkles, UserCheck, UserPlus, UserRound, Users } from '@lucide/vue'
import PostCard from '../components/PostCard.vue'
import { roleLabels, subjectLabels, trackLabels } from '../lib/labels'
import { sampleComments, samplePosts } from '../lib/sampleData'
import { useForumStore, type FollowProfile } from '../stores/forum'
import type { Comment, Post, Role } from '../types/forum'

const route = useRoute()
const router = useRouter()
const forumStore = useForumStore()

const profileName = computed(() => decodeURIComponent(String(route.params.name ?? '')))
const isCurrentUser = computed(() => forumStore.currentUser?.nickname === profileName.value)
const allPosts = computed(() => [...forumStore.getCreatedPosts(), ...samplePosts].map((post) => forumStore.hydratePost(post)))
const authoredPosts = computed(() => allPosts.value.filter((post) => post.authorName === profileName.value))
const allComments = computed(() => [
  ...Object.values(sampleComments).flat(),
  ...forumStore.getLocalComments(),
])
const authoredComments = computed(() => allComments.value.filter((comment) => comment.author === profileName.value))

const profile = computed(() => {
  const fromPost = authoredPosts.value[0]
  const fromComment = authoredComments.value[0]
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

const favoritePosts = computed(() => (isCurrentUser.value ? forumStore.getFavoritePosts(allPosts.value) : []))
const followingList = computed(() => forumStore.getFollowing(profile.value.name))
const followerList = computed(() => forumStore.getFollowers(profile.value.name))
const isFollowing = computed(() => forumStore.isUserFollowing(profile.value.name))
const profileAsFollow = computed<FollowProfile>(() => ({
  name: profile.value.name,
  role: profile.value.role,
  province: profile.value.province,
  grade: profile.value.grade,
  followedAt: new Date().toISOString(),
}))
const commentCards = computed(() =>
  authoredComments.value
    .map((comment) => {
      const post = allPosts.value.find((item) => item.id === comment.postId)
      return { comment, post }
    })
    .filter((item): item is { comment: Comment; post: Post } => Boolean(item.post)),
)

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
  forumStore.toggleUserFollow(profileAsFollow.value)
}
</script>

<template>
  <main class="detail-page user-profile-page">
    <button class="back-link" @click="router.back()"><ChevronLeft :size="17" /> 返回上一页</button>

    <section class="user-profile-hero">
      <div class="user-profile-avatar">{{ profile.name.slice(0, 1) }}</div>
      <div>
        <div class="breadcrumb">用户主页 / {{ roleTone(profile.role) }}</div>
        <h1>{{ profile.name }}</h1>
        <p>{{ profile.grade }} · {{ roleLabels[profile.role] }} · {{ profile.province }}</p>
        <div class="overview-metrics">
          <span><PenLine :size="17" /> {{ authoredPosts.length }} 篇帖子</span>
          <span><MessageSquare :size="17" /> {{ authoredComments.length }} 条评论</span>
          <span><UserPlus :size="17" /> {{ followingList.length }} 关注</span>
          <span><Users :size="17" /> {{ followerList.length }} 粉丝</span>
          <span v-if="isCurrentUser"><Bookmark :size="17" /> {{ favoritePosts.length }} 个收藏</span>
        </div>
      </div>
      <div class="user-profile-actions">
        <RouterLink v-if="isCurrentUser" class="write-button" to="/settings">
          <Settings :size="16" /> 编辑资料
        </RouterLink>
        <button v-else class="follow-button" :class="{ active: isFollowing }" type="button" @click="toggleFollow">
          <component :is="isFollowing ? UserCheck : UserPlus" :size="16" />
          {{ isFollowing ? '已关注' : '关注' }}
        </button>
      </div>
    </section>

    <section v-if="isCurrentUser" class="profile-choice-card">
      <div>
        <small><Sparkles :size="15" /> 我的选科画像</small>
        <h2>{{ trackLabels[forumStore.choiceProfile.preferredTrack] }} · {{ forumStore.choiceProfile.preferredSubjects.map((item) => subjectLabels[item]).join(' + ') }}</h2>
        <p>MBTI：{{ forumStore.choiceProfile.mbti || '未填写' }} · 目标专业：{{ forumStore.choiceProfile.targetMajors || '未填写' }}</p>
      </div>
      <RouterLink to="/settings">完善画像</RouterLink>
    </section>

    <section v-if="isCurrentUser" class="profile-section follow-management-section">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">我的关注</button>
        </div>
        <span class="profile-section-count">{{ followingList.length }} 人</span>
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

    <section v-if="isCurrentUser" class="profile-section follow-management-section">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">关注我的</button>
        </div>
        <span class="profile-section-count">{{ followerList.length }} 人</span>
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

    <section class="profile-section">
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

    <section class="profile-section">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">TA 的评论</button>
        </div>
      </div>
      <div v-if="commentCards.length" class="profile-comment-list">
        <RouterLink v-for="item in commentCards" :key="item.comment.id" :to="`/posts/${item.post.id}`">
          <span>{{ item.post.title }}</span>
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

    <section v-if="isCurrentUser" class="profile-section">
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
  </main>
</template>
