<script setup lang="ts">
import { computed, ref } from 'vue'
import { ChevronLeft, ExternalLink, FileText, MapPin, Newspaper, Search, ShieldCheck } from '@lucide/vue'
import { useRouter } from 'vue-router'
import {
  knowledgeTakeaways,
  mediaInsightSources,
  nationalOfficialSources,
  provinceKnowledge,
} from '../lib/knowledgeBase'

const router = useRouter()
const keyword = ref('')
const activeRegion = ref<'全部' | '华北' | '东北' | '华东' | '华中' | '华南' | '西南' | '西北' | '港澳台'>('全部')
const regions = ['全部', '华北', '东北', '华东', '华中', '华南', '西南', '西北', '港澳台'] as const

const filteredProvinces = computed(() => {
  const q = keyword.value.trim()
  return provinceKnowledge.filter((item) => {
    const matchRegion = activeRegion.value === '全部' || item.region === activeRegion.value
    const searchable = [
      item.province,
      item.region,
      item.authority,
      item.reformMode,
      item.status,
      item.focus.join(''),
      item.checklist.join(''),
    ].join('')
    return matchRegion && (!q || searchable.includes(q))
  })
})

function provincePath(province: string) {
  return `/knowledge/${encodeURIComponent(province)}`
}

function reformLabel(mode: string) {
  if (mode === '3+3') return '3+3 新高考'
  if (mode === '3+1+2') return '3+1+2 新高考'
  if (mode === 'special') return '特殊招生'
  return '传统/过渡'
}

function provinceTone(index: number) {
  return ['tone-mint', 'tone-blue', 'tone-amber', 'tone-rose', 'tone-violet'][index % 5]
}

const modeCount = computed(() => ({
  newGaokao: provinceKnowledge.filter((item) => item.reformMode === '3+3' || item.reformMode === '3+1+2').length,
  traditional: provinceKnowledge.filter((item) => item.reformMode === 'traditional').length,
  special: provinceKnowledge.filter((item) => item.reformMode === 'special').length,
  provinces: provinceKnowledge.length,
}))
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>

    <section class="knowledge-hero">
      <div>
        <div class="breadcrumb">工具 / 全国政策与舆情信息库</div>
        <h1>全国招生考试与选科知识库</h1>
        <p>把全国省级考试院入口、招生政策核对点、选考科目要求和媒体评论线索集中到一个页面。政策判断以官方来源为准，媒体内容用于理解趋势和用户焦虑。</p>
        <div class="overview-metrics">
          <span><ShieldCheck :size="18" /> {{ modeCount.provinces }} 个省级条目</span>
          <span>{{ modeCount.newGaokao }} 个新高考/改革省份</span>
          <span>{{ modeCount.traditional }} 个需按最新公告核对</span>
          <span>{{ modeCount.special }} 个港澳台特殊招生入口</span>
        </div>
      </div>
      <label class="knowledge-search">
        <Search :size="18" />
        <input v-model="keyword" placeholder="搜索省份、考试院、物化、志愿、专项计划..." />
      </label>
    </section>

    <nav class="content-lens-tabs" aria-label="地区筛选">
      <button
        v-for="region in regions"
        :key="region"
        type="button"
        :class="{ active: activeRegion === region }"
        @click="activeRegion = region"
      >
        {{ region }}
      </button>
    </nav>

    <section class="knowledge-source-grid">
      <article v-for="source in nationalOfficialSources" :key="source.url" class="knowledge-source-card">
        <small>官方来源</small>
        <h2>{{ source.title }}</h2>
        <p>{{ source.summary }}</p>
        <div class="mini-tag-row">
          <span v-for="tag in source.tags" :key="tag"># {{ tag }}</span>
        </div>
        <a :href="source.url" target="_blank" rel="noreferrer">
          {{ source.publisher }} <ExternalLink :size="14" />
        </a>
      </article>
    </section>

    <section class="province-knowledge-grid xhs-province-waterfall">
      <article
        v-for="(item, index) in filteredProvinces"
        :key="item.province"
        class="province-knowledge-card xhs-province-card"
      >
        <RouterLink class="province-card-main" :to="provincePath(item.province)">
          <div class="province-note-cover" :class="provinceTone(index)">
            <div>
              <small><MapPin :size="14" /> {{ item.region }}</small>
              <strong>{{ item.province }}</strong>
              <span>{{ reformLabel(item.reformMode) }}</span>
            </div>
            <em>{{ item.focus[0] }}</em>
          </div>
          <div class="province-note-body">
            <div class="province-card-head">
              <span>{{ item.authority }}</span>
              <small>{{ item.checklist.length }} 项核对</small>
            </div>
            <p>{{ item.status }}</p>
            <div class="mini-tag-row">
              <span v-for="tag in item.focus" :key="tag"># {{ tag }}</span>
            </div>
            <div class="province-checklist">
              <span v-for="task in item.checklist.slice(0, 3)" :key="task">
                <FileText :size="14" /> {{ task }}
              </span>
            </div>
          </div>
        </RouterLink>
        <footer class="province-note-footer">
          <RouterLink :to="provincePath(item.province)">查看资料包</RouterLink>
          <a :href="item.portalUrl" target="_blank" rel="noreferrer" @click.stop>
            官方入口 <ExternalLink :size="14" />
          </a>
        </footer>
      </article>
    </section>

    <section class="knowledge-review-band">
      <div>
        <h2>知识库使用原则</h2>
        <ul>
          <li v-for="item in knowledgeTakeaways" :key="item">{{ item }}</li>
        </ul>
      </div>
      <div class="media-source-stack">
        <article v-for="source in mediaInsightSources" :key="source.url">
          <small><Newspaper :size="14" /> {{ source.publisher }}</small>
          <h3>{{ source.title }}</h3>
          <p>{{ source.summary }}</p>
          <div class="mini-tag-row">
            <span v-for="tag in source.tags" :key="tag"># {{ tag }}</span>
          </div>
          <a :href="source.url" target="_blank" rel="noreferrer">查看原文 <ExternalLink :size="14" /></a>
        </article>
      </div>
    </section>
  </main>
</template>
