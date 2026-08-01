<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { Bookmark, ChevronLeft, MessageCircle, RefreshCcw, Search, ShieldCheck, SlidersHorizontal, Sparkles, ThumbsUp } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { fetchPostCollection, fetchPublishedRequirements } from '../lib/api'
import { formatCompactCount, getMajorForumStats, majorForumPath, toMajorRequirementCard } from '../lib/majorForum'
import { useForumStore } from '../stores/forum'
import { useOnlineState } from '../composables/useOnlineState'

const router = useRouter()
const forumStore = useForumStore()
const { isOffline } = useOnlineState()
const postsQuery = useQuery({
  queryKey: ['requirement-forum-posts'],
  queryFn: () => fetchPostCollection({ sort: 'latest', limit: 50 }),
})
const requirementsQuery = useQuery({
  queryKey: ['real-data', 'requirements'],
  queryFn: fetchPublishedRequirements,
})
const keyword = ref('')
const activeCategory = ref('全部')
const activeType = ref('全部')
const requirementRecords = computed(() => requirementsQuery.data.value ?? [])
const requirementCards = computed(() => requirementRecords.value.map(toMajorRequirementCard))
const categories = computed(() => ['全部', ...Array.from(new Set(requirementCards.value.map((item) => item.category)))])
const noteTypes = ['全部', '已复核要求', '需逐校核对']
const quickKeywords = ['临床医学', '计算机', '法学', '师范', '人工智能', '金融', '中医学', '电气工程']
const results = computed(() => {
  const q = keyword.value.trim()
  return requirementCards.value.filter((item) =>
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
  new Map(requirementCards.value.map((item) => [item.major, getMajorForumStats(item.major, forumStore, postsQuery.data.value ?? [])])),
)

const statsFor = (major: string) => statsByMajor.value.get(major) ?? getMajorForumStats(major, forumStore, [])

const goMajorForum = (major: string) => {
  router.push(majorForumPath(major))
}

function refetchRequirements() {
  void requirementsQuery.refetch()
  void postsQuery.refetch()
}

</script>

<template>
  <main class="detail-page requirements-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="requirements-xhs-hero">
      <div>
        <div class="breadcrumb">工具 / 选科要求查询</div>
        <h1>像刷笔记一样查专业选科</h1>
        <p>当前展示 {{ requirementCards.length }} 条已上架专业要求记录。广东优先展示已复核数据，其他地区明确标记待复核。</p>
        <div class="overview-metrics">
          <span><Sparkles :size="18" /> {{ requirementCards.length }} 条记录</span>
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

    <section v-if="requirementsQuery.isError.value || isOffline" class="empty-state">
      <RefreshCcw :size="28" />
      <h2>{{ isOffline ? '当前网络不可用' : '专业要求数据暂时不可用' }}</h2>
      <p>{{ isOffline ? '恢复网络后再重试，本页不会生成模拟专业要求。' : '请稍后重试；本页不会在接口失败时生成模拟结论。' }}</p>
      <button class="primary-wide compact" type="button" @click="refetchRequirements">重试</button>
    </section>

    <section v-else-if="!requirementsQuery.isLoading.value && !results.length" class="empty-state">
      <h2>暂无已复核数据</h2>
      <p>当前筛选条件下没有已上架专业要求记录，请调整关键词或等待管理员复核发布。</p>
    </section>

    <section v-else class="requirement-note-waterfall">
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
          <span>互动总数</span>
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
