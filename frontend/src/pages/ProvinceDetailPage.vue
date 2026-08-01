<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ChevronLeft, ExternalLink, FileText, RefreshCcw, ShieldCheck } from '@lucide/vue'
import PostCard from '../components/PostCard.vue'
import { fetchPostCollection, fetchProvinceCoverage, fetchPublishedPolicies } from '../lib/api'
import { policyDocumentPath } from '../lib/policyDocuments'
import { useOnlineState } from '../composables/useOnlineState'

const route = useRoute()
const router = useRouter()
const { isOffline } = useOnlineState()
const provinceName = computed(() => decodeURIComponent(String(route.params.province ?? '')))
const coverageQuery = useQuery({
  queryKey: ['real-data', 'provinces'],
  queryFn: fetchProvinceCoverage,
})
const policyRecordsQuery = useQuery({
  queryKey: ['real-data', 'policies', provinceName],
  queryFn: fetchPublishedPolicies,
})
const coverage = computed(() => coverageQuery.data.value?.find((item) => item.province === provinceName.value))
const hasCoverageRecord = computed(() => Boolean(coverage.value))
const provincePostsQuery = useQuery({
  queryKey: computed(() => ['province-posts', provinceName.value]),
  queryFn: () => fetchPostCollection({ province: provinceName.value, sort: 'latest', limit: 50 }),
  enabled: computed(() => hasCoverageRecord.value),
})
const nationalPostsQuery = useQuery({
  queryKey: ['province-posts', '全国'],
  queryFn: () => fetchPostCollection({ province: '全国', sort: 'latest', limit: 20 }),
})
const provincePosts = computed(() => {
  if (!hasCoverageRecord.value) return []
  const focused = provincePostsQuery.data.value ?? []
  const byPolicy = (nationalPostsQuery.data.value ?? []).filter((post) =>
    post.tags.some((tag) => tag.includes(provinceName.value) || provinceName.value.includes(tag)),
  )
  const merged = new Map([...focused, ...byPolicy].map((post) => [post.id, post]))
  return Array.from(merged.values()).slice(0, 12)
})

const fileCards = computed(() =>
  (policyRecordsQuery.data.value ?? [])
    .filter((record) =>
      record.coverageStatus === 'verified' &&
      (record.scope === provinceName.value || record.title.includes(provinceName.value)),
    )
    .slice(0, 12),
)
const hasProvinceError = computed(() => coverageQuery.isError.value || policyRecordsQuery.isError.value)

function refetchProvinceData() {
  void coverageQuery.refetch()
  void policyRecordsQuery.refetch()
  void provincePostsQuery.refetch()
  void nationalPostsQuery.refetch()
}
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/knowledge')"><ChevronLeft :size="17" /> 返回政策库</button>

    <section v-if="coverageQuery.isLoading.value" class="empty-state public-page-state">
      <RefreshCcw class="state-spin" :size="30" />
      <h1>正在加载省份资料</h1>
      <p>请稍等，正在同步 {{ provinceName }} 的 API 覆盖状态。</p>
    </section>

    <section v-else-if="hasProvinceError || isOffline" class="empty-state detail-empty-state public-page-state">
      <FileText :size="30" />
      <h1>{{ isOffline ? '当前网络不可用' : '省份资料加载失败' }}</h1>
      <p>{{ isOffline ? `恢复网络后再重试，当前不会推断 ${provinceName} 的数据状态。` : `服务暂时不可用，暂时不能确认 ${provinceName} 的已复核数据状态。` }}</p>
      <div class="state-action-row">
        <button class="primary-wide compact" type="button" @click="refetchProvinceData">重试</button>
        <RouterLink class="ghost-button compact" to="/knowledge">返回政策库</RouterLink>
      </div>
    </section>

    <template v-else>
    <section v-if="hasCoverageRecord" class="province-detail-hero">
      <div>
        <div class="breadcrumb">政策库 / API 覆盖状态 / {{ provinceName }}</div>
        <h1>{{ provinceName }}招生考试与选科文件</h1>
        <p>{{ coverage?.methodology || '暂无已复核方法说明。' }}</p>
        <div class="overview-metrics">
          <span><ShieldCheck :size="18" /> {{ coverage?.coverageStatus === 'verified' ? '已复核' : '暂无已复核数据' }}</span>
          <span>{{ coverage?.dataYear }} 年</span>
          <span>{{ coverage?.recordsCount }} 条记录</span>
        </div>
      </div>
      <span class="primary-wide compact">{{ coverage?.coverageStatus === 'verified' ? '可查看已复核文件' : '不展示模拟结论' }}</span>
    </section>

    <section v-if="hasCoverageRecord && fileCards.length" class="province-file-grid">
      <article v-for="file in fileCards" :key="file.id" class="province-file-card">
        <small><FileText :size="15" /> {{ file.type }}</small>
        <h2>{{ file.title }}</h2>
        <p>{{ file.summary || file.methodology }}</p>
        <div class="province-file-actions">
          <RouterLink :to="policyDocumentPath(provinceName, file.id)">
            <FileText :size="15" /> 查看网页化全文
          </RouterLink>
          <a :href="file.source.url || file.url" target="_blank" rel="noreferrer">
            官方/下载入口 <ExternalLink :size="14" />
          </a>
        </div>
      </article>
    </section>

    <section v-else-if="hasCoverageRecord && !policyRecordsQuery.isLoading.value" class="empty-state">
      <FileText :size="30" />
      <h2>暂无已复核政策文件</h2>
      <p>当前省份还没有通过 API 发布的已复核政策记录，本页不会生成模板文件。</p>
    </section>

    <section v-if="hasCoverageRecord" class="province-content-grid">
      <article>
        <h2>数据覆盖说明</h2>
        <ul>
          <li v-if="coverage?.coverageStatus === 'verified'">{{ coverage.methodology }}</li>
          <li v-else>暂无已复核数据，本页不会生成本地模拟政策或专业要求结论。</li>
        </ul>
      </article>
      <article>
        <h2>下载前核对清单</h2>
        <ol>
          <li>先确认文件发布年份，优先使用本省考试院最新公告。</li>
          <li>选考目录只解决“能不能报”，仍需结合招生计划和高校章程。</li>
          <li>PDF、附件或压缩包下载后，建议用专业名称和院校名称双重搜索。</li>
          <li>港澳台/传统高考地区不要直接套用“3+1+2”专业组规则。</li>
        </ol>
      </article>
    </section>

    <section v-if="hasCoverageRecord" class="feed-panel province-post-panel">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">{{ provinceName }}相关笔记</button>
        </div>
        <button class="sort-button" type="button" @click="router.push('/')">回首页看更多</button>
      </div>
      <p class="province-post-intro">
        这里汇总同省份、同政策焦点和全国通用的选科讨论。每张帖子都可以直接点赞、收藏，点进详情页后可以发表评论。
      </p>
      <div class="feed-grid">
        <PostCard v-for="post in provincePosts" :key="post.id" :post="post" />
      </div>
    </section>

    <section v-else class="empty-state">
      <h2>没有找到该省份</h2>
      <p>当前 API 未返回该省份覆盖记录。请返回政策库，从已接入省份卡片进入。</p>
    </section>
    </template>
  </main>
</template>
