<script setup lang="ts">
import { CircleCheck, Dna, FlaskConical, Globe2, Landmark } from '@lucide/vue'
import { computed } from 'vue'
import { categoryLabels, subjectAccent, subjectLabels, trackLabels } from '../lib/labels'
import { useForumStore } from '../stores/forum'
import type { Category, Subject, Track } from '../types/forum'

const forumStore = useForumStore()

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
  { label: '物化生', count: '12.6k', color: '#0f9f7a' },
  { label: '物化地', count: '9.8k', color: '#2563eb' },
  { label: '物生地', count: '7.3k', color: '#38bdf8' },
  { label: '物化政', count: '6.1k', color: '#ef4444' },
  { label: '史政地', count: '5.7k', color: '#f97316' },
  { label: '史地生', count: '4.2k', color: '#f59e0b' },
])
</script>

<template>
  <aside class="filter-rail">
    <div class="panel-title-row">
      <h2>选科组合筛选</h2>
      <button class="ghost-link">重置</button>
    </div>

    <section class="filter-section">
      <h3>方向</h3>
      <div class="segmented-control">
        <button
          v-for="track in tracks"
          :key="track"
          :class="{ active: forumStore.filter.track === track }"
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
          @click="forumStore.setCategory(category)"
        >
          {{ category === 'all' ? '全部' : categoryLabels[category] }}
        </button>
      </div>
    </section>

    <section class="filter-section hot-list">
      <h3>热门组合</h3>
      <button v-for="combo in hotCombos" :key="combo.label" class="hot-row">
        <span class="hot-dot" :style="{ background: combo.color }"></span>
        <span>{{ combo.label }}</span>
        <strong>{{ combo.count }}</strong>
      </button>
    </section>

    <button class="outline-wide">查看全部组合</button>
  </aside>
</template>
