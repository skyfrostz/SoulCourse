<script setup lang="ts">
import { ChevronDown, CircleCheck, Dna, FlaskConical, Globe2, Landmark, PanelLeftClose, PanelLeftOpen, SlidersHorizontal } from '@lucide/vue'
import { computed, ref } from 'vue'
import { categoryLabels, subjectAccent, subjectLabels } from '../lib/labels'
import { useGlobalSearch } from '../composables/useGlobalSearch'
import { useForumStore } from '../stores/forum'
import type { Category, Subject, SubjectInsight, Track } from '../types/forum'

const forumStore = useForumStore()
const { runSearch } = useGlobalSearch()
const filterExpanded = ref(false)
const props = defineProps<{ collapsed: boolean; insights: SubjectInsight[] }>()
defineEmits<{ toggleCollapse: [] }>()

const tracks: Array<Track | 'all'> = ['all', 'physics', 'history']
const subjects: Subject[] = ['chemistry', 'biology', 'politics', 'geography']
const categories: Array<Category | 'all'> = ['all', 'experience', 'question', 'data']

const subjectIcons = {
  chemistry: FlaskConical,
  biology: Dna,
  politics: Landmark,
  geography: Globe2,
}

const trackLabel = computed(() =>
  forumStore.filter.track === 'all' ? '全部方向' : forumStore.filter.track === 'physics' ? '物理方向' : '历史方向',
)
const subjectsLabel = computed(() =>
  forumStore.filter.subjects.length
    ? forumStore.filter.subjects.map((subject) => subjectLabels[subject]).join(' + ')
    : '不限科目',
)
const categoryLabel = computed(() =>
  forumStore.filter.category === 'all' ? '全部内容' : categoryLabels[forumStore.filter.category],
)
const activeFilterCount = computed(() =>
  Number(forumStore.filter.track !== 'all') + forumStore.filter.subjects.length + Number(forumStore.filter.category !== 'all'),
)

const officialRequirements = computed(() => props.insights.slice(0, 6))
const requirementColors = ['#0f9f7a', '#2563eb', '#38bdf8', '#ef4444', '#f97316', '#f59e0b']

function resetActiveFilters() {
  if (forumStore.filter.keyword) {
    void runSearch('')
    return
  }
  forumStore.resetFilters()
}
</script>

<template>
  <aside class="filter-rail" :class="{ collapsed }">
    <button
      class="rail-collapse-button rail-reopen-button"
      type="button"
      aria-label="展开筛选栏"
      title="展开筛选栏"
      @click="$emit('toggleCollapse')"
    >
      <PanelLeftOpen :size="19" />
    </button>
    <div class="panel-title-row filter-panel-heading">
      <div>
        <span class="filter-kicker">精准筛选</span>
        <h2>找适合你的讨论</h2>
        <p>按方向、再选科目和内容类型缩小范围</p>
      </div>
      <div class="filter-title-actions">
        <button class="filter-toggle" type="button" :aria-expanded="filterExpanded" aria-controls="community-filter-body" @click="filterExpanded = !filterExpanded">
          <SlidersHorizontal :size="16" /> {{ filterExpanded ? '收起' : '展开筛选' }}
          <span v-if="activeFilterCount" class="filter-count">{{ activeFilterCount }}</span>
          <ChevronDown class="filter-chevron" :size="15" />
        </button>
        <button v-if="activeFilterCount" class="ghost-link" type="button" @click="resetActiveFilters">重置</button>
        <button class="rail-collapse-button" type="button" aria-label="收起筛选栏" title="收起筛选栏" @click="$emit('toggleCollapse')">
          <PanelLeftClose :size="17" />
        </button>
      </div>
    </div>

    <div class="filter-summary" aria-label="当前筛选条件">
      <span><small>方向</small><strong>{{ trackLabel }}</strong></span>
      <span><small>科目</small><strong>{{ subjectsLabel }}</strong></span>
      <span><small>类型</small><strong>{{ categoryLabel }}</strong></span>
    </div>

    <div id="community-filter-body" class="filter-body" :class="{ open: filterExpanded }">
    <section class="filter-section">
      <div class="filter-section-heading">
        <span>1</span>
        <div><h3>方向</h3><small>先选物理或历史方向</small></div>
      </div>
      <div class="segmented-control track-control">
        <button
          v-for="track in tracks"
          :key="track"
          :class="{ active: forumStore.filter.track === track }"
          :aria-pressed="forumStore.filter.track === track"
          @click="forumStore.setTrack(track)"
        >
          {{ track === 'all' ? '全部' : track === 'physics' ? '物理' : '历史' }}
        </button>
      </div>
    </section>

    <section class="filter-section">
      <div class="filter-section-heading">
        <span>2</span>
        <div><h3>再选科目</h3><small>最多选择两门科目</small></div>
      </div>
      <button
        v-for="subject in subjects"
        :key="subject"
        class="subject-row"
        :class="{ checked: forumStore.filter.subjects.includes(subject) }"
        :aria-pressed="forumStore.filter.subjects.includes(subject)"
        @click="forumStore.toggleSubject(subject)"
      >
        <span class="subject-check">
          <CircleCheck v-if="forumStore.filter.subjects.includes(subject)" :size="15" />
        </span>
        <component :is="subjectIcons[subject]" :size="18" :style="{ color: subjectAccent[subject] }" />
        <span>{{ subjectLabels[subject] }}</span>
      </button>
    </section>

    <section class="filter-section">
      <div class="filter-section-heading">
        <span>3</span>
        <div><h3>内容类型</h3><small>选择想查看的讨论形式</small></div>
      </div>
      <div class="category-stack">
        <button
          v-for="category in categories"
          :key="category"
          :class="{ active: forumStore.filter.category === category }"
          :aria-pressed="forumStore.filter.category === category"
          @click="forumStore.setCategory(category)"
        >
          {{ category === 'all' ? '全部' : categoryLabels[category] }}
        </button>
      </div>
    </section>

    <section class="filter-section hot-list">
      <h3>官方计划要求</h3>
      <RouterLink v-for="(insight, index) in officialRequirements" :key="insight.id" class="hot-row" :to="`/insights/${insight.id}`">
        <span class="hot-dot" :style="{ background: requirementColors[index % requirementColors.length] }"></span>
        <span>{{ insight.combination }}</span>
        <strong>{{ insight.heat }}个</strong>
      </RouterLink>
    </section>

    <RouterLink class="outline-wide" to="/insights">查看全部组合</RouterLink>
    <button class="filter-confirm-button" type="button" @click="filterExpanded = false">
      <CircleCheck :size="17" /> 确定并收起
    </button>
    </div>
  </aside>
</template>
