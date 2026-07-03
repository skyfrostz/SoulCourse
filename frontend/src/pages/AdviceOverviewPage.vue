<script setup lang="ts">
import { computed, ref } from 'vue'
import { Bookmark, ChevronLeft, Heart, MessageCircle, PenLine, Search, ShieldCheck, Sparkles, ThumbsUp } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { useAdviceEngagement } from '../composables/useAdviceEngagement'
import { adviceNotes } from '../lib/adviceNotes'
import { useForumStore } from '../stores/forum'

const router = useRouter()
const forumStore = useForumStore()
const adviceEngagement = useAdviceEngagement()

const keyword = ref('')
const activeLens = ref<'all' | 'coverage' | 'pressure' | 'risk' | 'timeline' | 'parent'>('all')
const lenses = [
  { value: 'all', label: '全部' },
  { value: 'coverage', label: '专业覆盖' },
  { value: 'pressure', label: '学习压力' },
  { value: 'risk', label: '避坑核对' },
  { value: 'timeline', label: '时间线' },
  { value: 'parent', label: '家长视角' },
] as const

const adviceList = computed(() => {
  const q = keyword.value.trim()
  return adviceNotes.filter((item) =>
    (activeLens.value === 'all' || item.lens === activeLens.value) &&
    (!q ||
      [item.title, item.combination, item.author, item.role, item.tags.join(''), item.summary]
        .some((value) => value.includes(q))),
  )
})

function openNote(id: string) {
  router.push(`/advice/${id}`)
}

function toggleLike(event: MouseEvent, note: (typeof adviceNotes)[number]) {
  event.stopPropagation()
  adviceEngagement.toggleLike(note.id, note.likes)
}

function toggleSave(event: MouseEvent, note: (typeof adviceNotes)[number]) {
  event.stopPropagation()
  if (!forumStore.requireAuth()) return
  adviceEngagement.toggleSave(note.id, note.saves)
}

function openComments(event: MouseEvent, note: (typeof adviceNotes)[number]) {
  event.stopPropagation()
  router.push(`/advice/${note.id}#comments`)
}
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="advice-xhs-hero">
      <div>
        <div class="breadcrumb">首页 / 精选建议</div>
        <h1>精选建议，也要像经验贴一样好刷</h1>
        <p>把组合适配、学习压力、专业覆盖、家长沟通和政策核对拆成可收藏的笔记。每张卡都给出来源、行动清单和可追问方向。</p>
        <div class="overview-metrics">
          <span><Sparkles :size="18" /> {{ adviceNotes.length }} 条精选笔记</span>
          <span><Heart :size="18" /> 适合收藏复看</span>
          <span><ShieldCheck :size="18" /> 来源可追溯</span>
        </div>
      </div>
      <div class="advice-hero-side">
        <label class="requirement-search">
          <Search :size="18" />
          <input v-model="keyword" placeholder="搜索：物化生、家长、法学、时间线..." />
        </label>
        <button class="primary-wide compact" @click="forumStore.openPublish('question')">
          <PenLine :size="16" /> 发布我的选科问题
        </button>
      </div>
    </section>

    <nav class="content-lens-tabs" aria-label="建议筛选">
      <button
        v-for="lens in lenses"
        :key="lens.value"
        type="button"
        :class="{ active: activeLens === lens.value }"
        @click="activeLens = lens.value"
      >
        {{ lens.label }}
      </button>
    </nav>

    <section class="advice-waterfall">
      <article
        v-for="note in adviceList"
        :key="note.id"
        class="advice-note-card"
        role="button"
        tabindex="0"
        @click="openNote(note.id)"
        @keydown.enter="openNote(note.id)"
      >
        <div class="advice-note-visual" :class="`tone-${note.cover}`">
          <span>{{ note.metricLabel }}</span>
          <strong>{{ note.metric }}</strong>
          <small>{{ note.combination }}</small>
        </div>
        <div class="advice-note-body">
          <small>{{ note.author }} · {{ note.role }}</small>
          <h2>{{ note.title }}</h2>
          <p>{{ note.summary }}</p>
          <div class="mini-tag-row">
            <span v-for="tag in note.tags" :key="tag"># {{ tag }}</span>
          </div>
          <ol>
            <li v-for="action in note.actions.slice(0, 3)" :key="action">{{ action }}</li>
          </ol>
          <a class="note-source-link" :href="note.sourceUrl" target="_blank" rel="noreferrer" @click.stop>
            <ShieldCheck :size="14" /> {{ note.source }}
          </a>
          <div class="note-social-row">
            <button
              type="button"
              :class="{ active: adviceEngagement.getStats(note.id, note.likes, note.saves).liked }"
              @click="toggleLike($event, note)"
            >
              <ThumbsUp :size="15" /> {{ adviceEngagement.getStats(note.id, note.likes, note.saves).likes }}
            </button>
            <button type="button" @click="openComments($event, note)">
              <MessageCircle :size="15" /> {{ adviceEngagement.getStats(note.id, note.likes, note.saves).comments }}
            </button>
            <button
              type="button"
              :class="{ active: adviceEngagement.getStats(note.id, note.likes, note.saves).saved }"
              @click="toggleSave($event, note)"
            >
              <Bookmark :size="15" /> {{ adviceEngagement.getStats(note.id, note.likes, note.saves).saves }}
            </button>
          </div>
        </div>
      </article>
    </section>
  </main>
</template>
