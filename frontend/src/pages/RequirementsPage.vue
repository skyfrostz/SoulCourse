<script setup lang="ts">
import { computed, ref } from 'vue'
import { Bookmark, ChevronLeft, MessageCircle, Search, ShieldCheck, SlidersHorizontal, Sparkles, ThumbsUp } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { majorRequirementCategories, majorRequirements, majorRequirementStats } from '../lib/majorRequirements'
import { formatCompactCount, getMajorForumStats, majorForumPath } from '../lib/majorForum'
import { samplePosts } from '../lib/sampleData'
import { useForumStore } from '../stores/forum'

const router = useRouter()
const forumStore = useForumStore()
const keyword = ref('')
const activeCategory = ref('全部')
const activeType = ref('全部')
const categories = computed(() => ['全部', ...majorRequirementCategories])
const noteTypes = ['全部', '高频刚需', '理工强约束', '人文社科', '交叉新兴', '需逐校核对']
const quickKeywords = ['临床医学', '计算机', '法学', '师范', '人工智能', '金融', '中医学', '电气工程']
const results = computed(() => {
  const q = keyword.value.trim()
  return majorRequirements.filter((item) =>
    (activeCategory.value === '全部' || item.category === activeCategory.value) &&
    (activeType.value === '全部' || item.noteType === activeType.value) &&
    (!q ||
      [item.major, item.category, item.suggestedCombination, item.requiredSubjects.join(''), item.noteType]
        .some((value) => value.includes(q))),
  )
})

const setKeyword = (value: string) => {
  keyword.value = value
}

const statsByMajor = computed(() =>
  new Map(majorRequirements.map((item) => [item.major, getMajorForumStats(item.major, forumStore, samplePosts)])),
)

const statsFor = (major: string) => statsByMajor.value.get(major) ?? getMajorForumStats(major, forumStore, samplePosts)

const goMajorForum = (major: string) => {
  router.push(majorForumPath(major))
}
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="requirements-xhs-hero">
      <div>
        <div class="breadcrumb">工具 / 选科要求查询</div>
        <h1>像刷笔记一样查专业选科</h1>
        <p>覆盖 {{ majorRequirementStats.total }} 个常见本科专业。先用专业名、门类和组合快速筛，再进入官方目录逐校核对。</p>
        <div class="overview-metrics">
          <span><Sparkles :size="18" /> {{ majorRequirementStats.total }} 个专业</span>
          <span><ShieldCheck :size="18" /> 官方口径可核对</span>
          <span><SlidersHorizontal :size="18" /> 分类筛选</span>
        </div>
      </div>
      <div class="requirement-search-panel">
        <label class="requirement-search">
          <Search :size="18" />
          <input v-model="keyword" placeholder="搜索：临床医学、计算机、法学、师范..." />
        </label>
        <div class="quick-query-row">
          <button v-for="item in quickKeywords" :key="item" type="button" @click="setKeyword(item)">{{ item }}</button>
        </div>
      </div>
    </section>

    <section class="requirement-filter-board">
      <div>
        <strong>{{ results.length }}</strong>
        <span>条专业笔记</span>
      </div>
      <div class="scroll-chip-row">
        <button
          v-for="item in categories"
          :key="item"
          type="button"
          :class="{ active: activeCategory === item }"
          @click="activeCategory = item"
        >
          {{ item }}
        </button>
      </div>
      <div class="scroll-chip-row compact">
        <button
          v-for="item in noteTypes"
          :key="item"
          type="button"
          :class="{ active: activeType === item }"
          @click="activeType = item"
        >
          {{ item }}
        </button>
      </div>
    </section>

    <section class="requirement-note-waterfall">
      <article
        v-for="(item, index) in results"
        :key="item.major"
        class="requirement-note-card"
        role="link"
        tabindex="0"
        @click="goMajorForum(item.major)"
        @keydown.enter="goMajorForum(item.major)"
        @keydown.space.prevent="goMajorForum(item.major)"
      >
        <div class="requirement-note-cover" :class="`tone-${index % 5}`">
          <small>{{ item.noteType }}</small>
          <strong>{{ formatCompactCount(statsFor(item.major).hotScore) }}</strong>
          <span>论坛热度</span>
          <div class="cover-bars" aria-hidden="true">
            <i :style="{ height: `${36 + (index % 4) * 10}px` }"></i>
            <i :style="{ height: `${54 + (index % 3) * 12}px` }"></i>
            <i :style="{ height: `${30 + (index % 5) * 8}px` }"></i>
          </div>
        </div>
        <div class="requirement-note-body">
          <small>{{ item.category }}</small>
          <h2>{{ item.major }}</h2>
          <div class="requirement-subjects">
            <span v-for="subject in item.requiredSubjects" :key="subject">{{ subject }}</span>
          </div>
          <p><strong>建议组合：</strong>{{ item.suggestedCombination }}</p>
          <p>{{ item.risk }}</p>
          <div class="mini-tag-row">
            <span># {{ item.noteType }}</span>
            <span># {{ item.category }}</span>
            <span># 逐校核对</span>
          </div>
          <footer>
            <a :href="item.sourceUrl" target="_blank" rel="noreferrer" @click.stop><ShieldCheck :size="15" /> {{ item.source }}</a>
          </footer>
          <div class="note-social-row">
            <span><Sparkles :size="15" /> {{ statsFor(item.major).postCount }} 篇</span>
            <span><ThumbsUp :size="15" /> {{ formatCompactCount(statsFor(item.major).likesCount) }}</span>
            <span><MessageCircle :size="15" /> {{ formatCompactCount(statsFor(item.major).commentsCount) }}</span>
            <span><Bookmark :size="15" /> {{ formatCompactCount(statsFor(item.major).favoritesCount) }}</span>
          </div>
        </div>
      </article>
    </section>
  </main>
</template>
