<script setup lang="ts">
import { ChevronDown, CircleCheck, Dna, FlaskConical, Globe2, Landmark, PanelLeftClose, PanelLeftOpen, SlidersHorizontal } from '@lucide/vue'
import { computed, ref } from 'vue'
import { categoryLabels, subjectAccent, subjectLabels, trackLabels } from '../lib/labels'
import { useForumStore } from '../stores/forum'
import type { Category, Subject, Track } from '../types/forum'

const forumStore = useForumStore()
const filterExpanded = ref(false)
defineProps<{ collapsed: boolean }>()
defineEmits<{ toggleCollapse: [] }>()

const tracks: Track[] = ['physics', 'history']
const subjects: Subject[] = ['chemistry', 'biology', 'politics', 'geography']
const categories: Array<Category | 'all'> = ['all', 'experience', 'question', 'data']

const subjectIcons = {
  chemistry: FlaskConical,
  biology: Dna,
  politics: Landmark,
  geography: Globe2,
}

const hotCombos = computed(() => [
  { label: '物化生', count: '12.6k', color: '#0f9f7a', track: 'physics' as Track, subjects: ['chemistry', 'biology'] as Subject[] },
  { label: '物化地', count: '9.8k', color: '#2563eb', track: 'physics' as Track, subjects: ['chemistry', 'geography'] as Subject[] },
  { label: '物生地', count: '7.3k', color: '#38bdf8', track: 'physics' as Track, subjects: ['biology', 'geography'] as Subject[] },
  { label: '物化政', count: '6.1k', color: '#ef4444', track: 'physics' as Track, subjects: ['chemistry', 'politics'] as Subject[] },
  { label: '史政地', count: '5.7k', color: '#f97316', track: 'history' as Track, subjects: ['politics', 'geography'] as Subject[] },
  { label: '史地生', count: '4.2k', color: '#f59e0b', track: 'history' as Track, subjects: ['geography', 'biology'] as Subject[] },
])

function selectCombo(combo: { label: string; track: Track; subjects: Subject[] }) {
  forumStore.setTrack(combo.track)
  forumStore.setSubjects(combo.subjects)
  forumStore.setKeyword(combo.label)
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
    <div class="panel-title-row">
      <h2>选科组合筛选</h2>
      <div class="filter-title-actions">
        <button class="filter-toggle" type="button" :aria-expanded="filterExpanded" aria-controls="community-filter-body" @click="filterExpanded = !filterExpanded">
          <SlidersHorizontal :size="15" /> 筛选 <ChevronDown :size="14" />
        </button>
        <button class="ghost-link" type="button" @click="forumStore.resetFilters()">重置</button>
        <button class="rail-collapse-button" type="button" aria-label="收起筛选栏" title="收起筛选栏" @click="$emit('toggleCollapse')">
          <PanelLeftClose :size="17" />
        </button>
      </div>
    </div>

    <div id="community-filter-body" class="filter-body" :class="{ open: filterExpanded }">
    <section class="filter-section">
      <h3>方向</h3>
      <div class="segmented-control">
        <button
          v-for="track in tracks"
          :key="track"
          :class="{ active: forumStore.filter.track === track }"
          :aria-pressed="forumStore.filter.track === track"
          @click="forumStore.setTrack(track)"
        >
          {{ trackLabels[track] }}
        </button>
      </div>
    </section>

    <section class="filter-section">
      <h3>再选科目</h3>
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
      <h3>内容类型</h3>
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
      <h3>热门组合</h3>
      <button v-for="combo in hotCombos" :key="combo.label" class="hot-row" type="button" @click="selectCombo(combo)">
        <span class="hot-dot" :style="{ background: combo.color }"></span>
        <span>{{ combo.label }}</span>
        <strong>{{ combo.count }}</strong>
      </button>
    </section>

    <RouterLink class="outline-wide" to="/insights">查看全部组合</RouterLink>
    </div>
  </aside>
</template>
