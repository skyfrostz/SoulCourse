<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { ChevronLeft, ExternalLink, FileText, MapPin, RefreshCcw, Search, ShieldCheck } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { apiDataEnabled, fetchProvinceCoverage, fetchPublishedPolicies, type ProvinceCoverage, type RealDataRecord } from '../lib/api'
import { knowledgeTakeaways } from '../lib/knowledgeBase'
import { useOnlineState } from '../composables/useOnlineState'

const router = useRouter()
const { isOffline } = useOnlineState()
const keyword = ref('')
const activeRegion = ref<'全部' | '已复核' | '待复核'>('全部')
const regions = ['全部', '已复核', '待复核'] as const
const provinceCoverageQuery = useQuery({
  queryKey: ['real-data', 'provinces'],
  queryFn: fetchProvinceCoverage,
  enabled: apiDataEnabled,
})
const policyRecordsQuery = useQuery({
  queryKey: ['real-data', 'policies'],
  queryFn: fetchPublishedPolicies,
  enabled: apiDataEnabled,
})
const hasKnowledgeError = computed(() => provinceCoverageQuery.isError.value || policyRecordsQuery.isError.value)
const isKnowledgeLoading = computed(() => provinceCoverageQuery.isLoading.value || policyRecordsQuery.isLoading.value)
const verifiedSourceRecords = computed(() => {
  const seen = new Set<string>()
  return (policyRecordsQuery.data.value ?? []).filter((record) => {
    if (record.coverageStatus !== 'verified' || !record.source.url || seen.has(record.source.url)) return false
    seen.add(record.source.url)
    return true
  }).slice(0, 8)
})

const filteredProvinces = computed(() => {
  const q = keyword.value.trim()
  return (provinceCoverageQuery.data.value ?? []).filter((item) => {
    const matchRegion =
      activeRegion.value === '全部' ||
      (activeRegion.value === '已复核' && item.coverageStatus === 'verified') ||
      (activeRegion.value === '待复核' && item.coverageStatus !== 'verified')
    const searchable = [
      item.province,
      item.coverageStatus,
      item.methodology,
      String(item.dataYear),
    ].join('')
    return matchRegion && (!q || searchable.includes(q))
  })
})

function provincePath(province: string) {
  return `/knowledge/${encodeURIComponent(province)}`
}

function coverageLabel(status: ProvinceCoverage['coverageStatus']) {
  return status === 'verified' ? '已复核数据' : '暂无已复核数据'
}

function provinceTone(index: number) {
  return ['tone-mint', 'tone-blue', 'tone-amber', 'tone-rose', 'tone-violet'][index % 5]
}

const modeCount = computed(() => ({
  verified: (provinceCoverageQuery.data.value ?? []).filter((item) => item.coverageStatus === 'verified').length,
  pending: (provinceCoverageQuery.data.value ?? []).filter((item) => item.coverageStatus !== 'verified').length,
  provinces: provinceCoverageQuery.data.value?.length ?? 0,
}))

const policyImagesByScope = computed(() => {
  const map = new Map<string, string>()
  ;(policyRecordsQuery.data.value ?? []).forEach((record) => {
    const image = firstRecordImage(record)
    if (!image) return
    if (record.scope) map.set(record.scope, image)
  })
  return map
})

const policyImagesByUrl = computed(() => {
  const map = new Map<string, string>()
  ;(policyRecordsQuery.data.value ?? []).forEach((record) => {
    const image = firstRecordImage(record)
    if (image && record.url) map.set(record.url, image)
  })
  return map
})

function firstRecordImage(record: RealDataRecord) {
  void record
  return ''
}

function provinceCover(province: string) {
  return policyImagesByScope.value.get(province) || ''
}

function sourceCover(url: string) {
  return policyImagesByUrl.value.get(url) || ''
}

function refetchKnowledge() {
  void provinceCoverageQuery.refetch()
  void policyRecordsQuery.refetch()
}
</script>

<template>
  <main class="detail-page knowledge-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>

    <section class="knowledge-hero">
      <div>
        <div class="breadcrumb">工具 / 全国政策与舆情信息库</div>
        <h1>全国招生考试与选科知识库</h1>
        <p>把已接入 API 的省份覆盖状态、招生政策、选考科目要求和来源线索集中到一个页面。政策判断以官方来源和已复核记录为准。</p>
        <div class="overview-metrics">
          <span><ShieldCheck :size="18" /> {{ modeCount.provinces }} 个省级条目</span>
          <span>{{ modeCount.verified }} 个已复核</span>
          <span>{{ modeCount.pending }} 个暂无已复核数据</span>
        </div>
      </div>
      <label class="knowledge-search">
        <Search :size="18" />
        <input v-model="keyword" placeholder="搜索省份、年份、方法说明..." />
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

    <section v-if="!isKnowledgeLoading && !hasKnowledgeError && verifiedSourceRecords.length" class="knowledge-source-grid">
      <article v-for="record in verifiedSourceRecords" :key="record.id" class="knowledge-source-card">
        <img v-if="sourceCover(record.source.url)" class="knowledge-source-cover" :src="sourceCover(record.source.url)" :alt="record.title" />
        <small>官方来源</small>
        <h2>{{ record.title }}</h2>
        <p>{{ record.summary || record.methodology }}</p>
        <div class="mini-tag-row">
          <span v-for="tag in record.tags" :key="tag"># {{ tag }}</span>
        </div>
        <a :href="record.source.url" target="_blank" rel="noreferrer">
          {{ record.source.name }} <ExternalLink :size="14" />
        </a>
      </article>
    </section>

    <section v-if="hasKnowledgeError || isOffline" class="empty-state public-page-state">
      <RefreshCcw :size="30" />
      <h2>{{ isOffline ? '当前网络不可用' : '政策资料加载失败' }}</h2>
      <p>{{ isOffline ? '恢复网络后再重试，已复核数据不会被模拟内容替代。' : '服务暂时不可用，已复核数据没有同步成功。请重试后再查看政策结论。' }}</p>
      <div class="state-action-row">
        <button class="primary-wide compact" type="button" @click="refetchKnowledge">重试</button>
      </div>
    </section>

    <section v-else-if="isKnowledgeLoading" class="empty-state public-page-state">
      <RefreshCcw class="state-spin" :size="30" />
      <h2>正在加载政策资料</h2>
      <p>请稍等，正在同步省份覆盖状态和已复核政策记录。</p>
    </section>

    <section v-else class="province-knowledge-grid xhs-province-waterfall">
      <article
        v-for="(item, index) in filteredProvinces"
        :key="item.province"
        class="province-knowledge-card xhs-province-card"
      >
        <RouterLink class="province-card-main" :to="provincePath(item.province)">
          <div class="province-note-cover" :class="[provinceTone(index), { 'has-image': provinceCover(item.province) }]">
            <img
              v-if="provinceCover(item.province)"
              class="province-note-image"
              :src="provinceCover(item.province)"
              :alt="`${item.province}资料包图片`"
            />
            <div>
              <small><MapPin :size="14" /> {{ item.coverageStatus === 'verified' ? '官方已复核' : '待复核' }}</small>
              <strong>{{ item.province }}</strong>
              <span>{{ coverageLabel(item.coverageStatus) }}</span>
            </div>
            <em>{{ item.dataYear }} 年</em>
          </div>
          <div class="province-note-body">
            <div class="province-card-head">
              <span>{{ item.coverageStatus === 'verified' ? '已发布结构化数据' : '不展示模拟结论' }}</span>
              <small>{{ item.recordsCount }} 条记录</small>
            </div>
            <p>{{ item.methodology || '暂无已复核数据，页面不会生成本地模拟政策或专业要求结论。' }}</p>
            <div class="mini-tag-row">
              <span># {{ item.coverageStatus === 'verified' ? '已复核' : '待复核' }}</span>
              <span># {{ item.dataYear }}</span>
            </div>
            <div class="province-checklist">
              <span><FileText :size="14" /> 仅展示 API 返回的覆盖状态和已复核记录</span>
            </div>
          </div>
        </RouterLink>
        <footer class="province-note-footer">
          <RouterLink :to="provincePath(item.province)">查看资料包</RouterLink>
          <span>{{ item.coverageStatus === 'verified' ? '可查看已复核文件' : '暂无已复核文件' }}</span>
        </footer>
      </article>
    </section>

    <section v-if="!isKnowledgeLoading && !hasKnowledgeError && !filteredProvinces.length" class="empty-state">
      <FileText :size="30" />
      <h2>暂无匹配的已复核数据</h2>
      <p>请调整筛选；本页不会根据本地静态清单生成省份结论。</p>
    </section>

    <section class="knowledge-review-band">
      <div>
        <h2>知识库使用原则</h2>
        <ul>
          <li v-for="item in knowledgeTakeaways" :key="item">{{ item }}</li>
        </ul>
      </div>
      <div class="media-source-stack">
        <article>
          <small><FileText :size="14" /> 数据边界</small>
          <h3>只发布可追溯记录</h3>
          <p>政策标题、摘要、年份、适用范围、采集方法和来源链接均来自已发布 API；缺少复核记录时明确显示空状态。</p>
        </article>
      </div>
    </section>
  </main>
</template>
