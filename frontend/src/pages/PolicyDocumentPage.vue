<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { BookOpenCheck, ChevronLeft, ExternalLink, FileText, Search, ShieldCheck } from '@lucide/vue'
import { fetchPublishedPolicies } from '../lib/api'
import { policyDocumentPath } from '../lib/policyDocuments'
import { useGlobalSearch } from '../composables/useGlobalSearch'
import { useOnlineState } from '../composables/useOnlineState'

const route = useRoute()
const router = useRouter()
const { runSearch } = useGlobalSearch()
const { isOffline } = useOnlineState()
const provinceName = computed(() => decodeURIComponent(String(route.params.province ?? '')))
const documentId = computed(() => String(route.params.documentId ?? ''))
const policiesQuery = useQuery({
  queryKey: ['real-data', 'policies', provinceName],
  queryFn: fetchPublishedPolicies,
})
const documents = computed(() =>
  (policiesQuery.data.value ?? [])
    .filter((record) =>
      record.coverageStatus === 'verified' &&
      (record.scope === provinceName.value || record.title.includes(provinceName.value)),
    ),
)
const document = computed(() => documents.value.find((item) => item.id === documentId.value))
const documentTags = computed(() => document.value?.tags.slice(0, 4) ?? [])
const selectedFileName = computed(() => String(route.query.file ?? ''))
const selectedFile = computed(() => document.value?.localDocuments?.find((file) => file.name === selectedFileName.value) ?? document.value?.localDocuments?.[0])
const selectedFileKind = computed(() => {
  const name = selectedFile.value?.name.toLowerCase() ?? ''
  if (name.endsWith('.pdf')) return 'pdf'
  if (/\.(doc|docx|xls|xlsx)$/.test(name)) return 'office'
  if (/\.(html|htm)$/.test(name)) return 'html'
  return 'download'
})
const selectedFileURL = computed(() => {
  if (!selectedFile.value) return ''
  const url = new URL(selectedFile.value.url, window.location.origin)
  url.searchParams.set('inline', '1')
  return url.toString()
})
const officeViewerURL = computed(() => `https://view.officeapps.live.com/op/view.aspx?src=${encodeURIComponent(selectedFileURL.value)}`)

function goBack() {
  router.push(provinceName.value ? `/knowledge/${encodeURIComponent(provinceName.value)}` : '/knowledge')
}

function searchInForum(query: string) {
  void runSearch(query)
}
</script>

<template>
  <main class="detail-page policy-document-page">
    <button class="back-link" @click="goBack"><ChevronLeft :size="17" /> 返回省份资料包</button>

    <section v-if="document" class="policy-document-hero">
      <div>
        <div class="breadcrumb">政策库 / {{ provinceName }} / 来源记录</div>
        <h1>{{ document.title }}</h1>
        <p>{{ document.summary || document.methodology }}</p>
        <div class="overview-metrics">
          <span><ShieldCheck :size="18" /> {{ document.source.name }}</span>
          <span>{{ document.dataYear }}</span>
          <span v-for="tag in documentTags" :key="tag"># {{ tag }}</span>
        </div>
      </div>
      <div class="policy-document-actions">
        <a :href="document.source.url || document.url" target="_blank" rel="noreferrer" class="primary-wide compact">
          官方来源 <ExternalLink :size="15" />
        </a>
      </div>
    </section>

    <section v-if="document" class="policy-document-layout">
      <aside class="policy-document-toc">
        <strong><BookOpenCheck :size="17" /> {{ provinceName }}已复核记录</strong>
        <RouterLink
          v-for="item in documents"
          :key="item.id"
          :to="policyDocumentPath(provinceName, item.id)"
          :class="{ active: item.id === document.id }"
        >
          <small>{{ item.type }}</small>
          <span>{{ item.title }}</span>
        </RouterLink>
      </aside>

      <article class="policy-document-reader">
        <section v-if="selectedFile" class="policy-file-viewer">
          <div class="policy-file-viewer-head">
            <div>
              <strong>{{ selectedFile.name }}</strong>
              <small>{{ selectedFile.type }} · {{ Math.ceil(selectedFile.sizeBytes / 1024) }} KB</small>
            </div>
            <a :href="`${selectedFile.url}?download=1`" class="ghost-button compact"><FileText :size="15" /> 下载原文件</a>
          </div>
          <iframe v-if="selectedFileKind === 'pdf'" :src="selectedFileURL" :title="`${selectedFile.name}在线预览`" class="policy-file-frame" />
          <iframe v-else-if="selectedFileKind === 'office'" :src="officeViewerURL" :title="`${selectedFile.name}在线预览`" class="policy-file-frame" />
          <iframe v-else-if="selectedFileKind === 'html'" :src="selectedFileURL" :title="`${selectedFile.name}在线预览`" class="policy-file-frame policy-html-frame" sandbox="allow-same-origin" />
          <div v-else class="policy-file-unavailable">
            <FileText :size="22" />
            <p>该文件格式暂不支持网页内预览，请下载原文件查看。</p>
          </div>
          <div v-if="(document.localDocuments?.length ?? 0) > 1" class="policy-file-switcher" aria-label="选择政策文件">
            <RouterLink
              v-for="file in document.localDocuments"
              :key="file.url"
              :to="{ path: policyDocumentPath(provinceName, document.id), query: { file: file.name } }"
              :class="{ active: file.name === selectedFile.name }"
            >
              {{ file.type }} · {{ file.name }}
            </RouterLink>
          </div>
        </section>

        <section class="policy-reader-note">
          <FileText :size="20" />
          <div>
            <strong>来源记录摘要</strong>
            <p>
              本页只展示已发布来源记录的摘要、采集方法和校验信息，不生成政策原文。最终报考仍以官方 PDF、考试院公告和高校招生章程为准。
            </p>
          </div>
        </section>

        <section class="policy-reader-section">
          <h2>记录摘要</h2>
          <p>{{ document.summary || '暂无摘要，请打开官方来源核对完整内容。' }}</p>
          <ul>
            <li>覆盖状态：{{ document.coverageStatus === 'verified' ? '已复核' : '暂无已复核数据' }}</li>
            <li>适用范围：{{ document.scope }}</li>
            <li>采集时间：{{ document.capturedAt }}</li>
            <li>文件哈希：{{ document.fileHash }}</li>
          </ul>
        </section>


        <section class="policy-reader-section">
          <h2>方法说明</h2>
          <p>{{ document.methodology || '管理员尚未补充方法说明。' }}</p>
        </section>

        <section class="policy-check-board">
          <div>
            <h2>下载前核对清单</h2>
            <ol>
              <li>确认官方来源域名、发布单位和文件年份。</li>
              <li>核对记录哈希与管理员采集说明是否完整。</li>
              <li>专业选考、招生计划和高校章程需要分别查证。</li>
            </ol>
          </div>
          <div>
            <h2>站内继续检索</h2>
            <button
              v-for="query in [provinceName, document.title, document.type]"
              :key="query"
              type="button"
              @click="searchInForum(query)"
            >
              <Search :size="15" /> {{ query }}
            </button>
          </div>
        </section>
      </article>
    </section>

    <section v-else-if="policiesQuery.isError.value || isOffline" class="empty-state">
      <FileText :size="30" />
      <h1>{{ isOffline ? '当前网络不可用' : '政策记录暂时无法加载' }}</h1>
      <p>{{ isOffline ? '恢复网络后再重试，当前不会推断这条记录的复核状态。' : '服务暂时不可用，当前无法确认这条记录的复核状态。' }}</p>
      <button class="primary-wide compact" type="button" @click="policiesQuery.refetch()">重新加载</button>
    </section>

    <section v-else class="empty-state">
      <FileText :size="30" />
      <h1>{{ policiesQuery.isLoading.value ? '正在加载政策记录' : '没有找到已复核政策记录' }}</h1>
      <p>请返回政策库，从省份资料包重新进入；本站不会为缺失来源生成模板正文。</p>
      <button class="primary-wide compact" type="button" @click="router.push('/knowledge')">返回政策库</button>
    </section>
  </main>
</template>
