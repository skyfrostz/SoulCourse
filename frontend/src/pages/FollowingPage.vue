<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ChevronLeft, Search, UserPlus, Users } from '@lucide/vue'
import { roleLabels } from '../lib/labels'
import { useForumStore, type FollowProfile } from '../stores/forum'

const router = useRouter()
const forumStore = useForumStore()
const activeTab = ref<'following' | 'followers'>('following')
const keyword = ref('')

const currentName = computed(() => forumStore.currentUser?.nickname ?? '')
const followingList = computed(() => currentName.value ? forumStore.getFollowing(currentName.value) : [])
const followerList = computed(() => currentName.value ? forumStore.getFollowers(currentName.value) : [])
const activeList = computed(() => activeTab.value === 'following' ? followingList.value : followerList.value)
const filteredUsers = computed(() => {
  const q = keyword.value.trim()
  if (!q) return activeList.value
  return activeList.value.filter((user) =>
    [user.name, user.province, user.grade, roleLabels[user.role]].some((value) => value.includes(q)),
  )
})

function userPath(user: FollowProfile) {
  return `/users/${encodeURIComponent(user.name)}`
}
</script>

<template>
  <main class="detail-page following-page">
    <button class="back-link" @click="router.back()"><ChevronLeft :size="17" /> 返回上一页</button>

    <section class="following-hero">
      <div>
        <div class="breadcrumb">个人中心 / 关注关系</div>
        <h1>我的关注</h1>
        <p>把你关注的老师、规划师、学长学姐和同组选科同学集中管理，后续可以从这里快速进入主页继续追问。</p>
        <div class="overview-metrics">
          <span><UserPlus :size="17" /> {{ followingList.length }} 个关注</span>
          <span><Users :size="17" /> {{ followerList.length }} 个粉丝</span>
        </div>
      </div>
      <label class="knowledge-search">
        <Search :size="18" />
        <input v-model="keyword" placeholder="搜索姓名、身份、省份..." />
      </label>
    </section>

    <nav class="content-lens-tabs following-tabs" aria-label="关注筛选">
      <button type="button" :class="{ active: activeTab === 'following' }" @click="activeTab = 'following'">
        我的关注 {{ followingList.length }}
      </button>
      <button type="button" :class="{ active: activeTab === 'followers' }" @click="activeTab = 'followers'">
        关注我的 {{ followerList.length }}
      </button>
    </nav>

    <section v-if="forumStore.currentUser" class="following-user-grid">
      <RouterLink v-for="user in filteredUsers" :key="user.name" class="following-user-card" :to="userPath(user)">
        <span class="user-profile-avatar compact">{{ user.name.slice(0, 1) }}</span>
        <span>
          <strong>{{ user.name }}</strong>
          <small>{{ user.grade }} · {{ roleLabels[user.role] }} · {{ user.province }}</small>
        </span>
        <em>{{ activeTab === 'following' ? '进入主页' : '查看粉丝' }}</em>
      </RouterLink>
    </section>

    <section v-if="forumStore.currentUser && !filteredUsers.length" class="empty-state compact-empty">
      <Users :size="30" />
      <h2>{{ activeTab === 'following' ? '还没有关注用户' : '还没有人关注你' }}</h2>
      <p>{{ activeTab === 'following' ? '在用户主页或帖子作者区点击关注，会出现在这里。' : '多发布经验帖、认真回复评论，更容易获得关注。' }}</p>
    </section>

    <section v-if="!forumStore.currentUser" class="empty-state compact-empty">
      <Users :size="30" />
      <h2>登录后查看关注关系</h2>
      <p>关注、粉丝和收藏会和你的账号一起保存。</p>
      <button class="primary-wide compact" type="button" @click="forumStore.authOpen = true">登录 / 注册</button>
    </section>
  </main>
</template>
