<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { BookOpenCheck, CheckCircle2, ChevronLeft, Download, ExternalLink, FileText, RefreshCcw } from '@lucide/vue'
import {
  fetchPublishedPolicies,
  getPolicyExamGroup,
  policyDisplayName,
  policyExamGroupLabels,
  sortPolicyRecords,
  type LocalPolicyDocument,
  type PolicyExamGroup,
} from '../lib/api'
import { policyDocumentPath } from '../lib/policyDocuments'
import { useOnlineState } from '../composables/useOnlineState'

const route = useRoute()
const router = useRouter()
const { isOffline } = useOnlineState()
const provinceName = computed(() => decodeURIComponent(String(route.params.province ?? '')))
const documentId = computed(() => String(route.params.documentId ?? ''))
const policiesQuery = useQuery({
  queryKey: ['real-data', 'policies', provinceName],
  queryFn: fetchPublishedPolicies,
})
const documents = computed(() => sortPolicyRecords(
  (policiesQuery.data.value ?? [])
    .filter((record) =>
      record.coverageStatus === 'verified' &&
      (record.scope === provinceName.value || record.title.includes(provinceName.value)),
    ),
))
const document = computed(() => documents.value.find((item) => item.id === documentId.value))
const documentGroups = computed(() => {
  const groups = new Map<PolicyExamGroup, typeof documents.value>()
  for (const item of documents.value) {
    const group = getPolicyExamGroup(item)
    const items = groups.get(group) ?? []
    items.push(item)
    groups.set(group, items)
  }
  return (['ordinary', 'art', 'sports', 'adult', 'other'] as PolicyExamGroup[])
    .filter((group) => groups.has(group))
    .map((group) => ({ group, label: policyExamGroupLabels[group], items: groups.get(group) ?? [] }))
})
const documentTags = computed(() => document.value?.tags.slice(0, 4) ?? [])
const localFileGroups = computed(() => {
  const groups = new Map<PolicyExamGroup, LocalPolicyDocument[]>()
  for (const file of sortPolicyRecords(document.value?.localDocuments ?? [])) {
    const group = getPolicyExamGroup(file)
    const items = groups.get(group) ?? []
    items.push(file)
    groups.set(group, items)
  }
  return (['ordinary', 'art', 'sports', 'adult', 'other'] as PolicyExamGroup[])
    .filter((group) => groups.has(group))
    .map((group) => ({ group, label: policyExamGroupLabels[group], items: groups.get(group) ?? [] }))
})
const selectedFileName = computed(() => String(route.query.file ?? ''))
const selectedFile = computed(() => document.value?.localDocuments?.find((file) =>
  file.name === selectedFileName.value || file.displayName === selectedFileName.value,
) ?? document.value?.localDocuments?.[0])
const selectedFileDisplayName = computed(() => selectedFile.value ? policyDisplayName(selectedFile.value) : '')
const selectedFileKind = computed(() => {
  const name = selectedFile.value?.name.toLowerCase() ?? ''
  if (name.endsWith('.pdf')) return 'pdf'
  if (/\.(doc|docx|xls|xlsx)$/.test(name)) return 'office'
  if (/\.(html|htm)$/.test(name)) return 'html'
  return 'download'
})
const htmlPreview = ref('')
const htmlPreviewLoading = ref(false)
const htmlPreviewError = ref(false)
const detectedPreviewKind = ref<'html' | 'pdf'>('html')
const selectedFileURL = computed(() => {
  if (!selectedFile.value) return ''
  const url = new URL(selectedFile.value.url, window.location.origin)
  // Version the preview request so previously cached DENY headers cannot break iframes.
  url.searchParams.set('preview', '2')
  return url.toString()
})
const officeViewerURL = computed(() => `https://view.officeapps.live.com/op/view.aspx?src=${encodeURIComponent(selectedFileURL.value)}`)
const downloadFileURL = computed(() => {
  if (!selectedFile.value) return ''
  const url = new URL(selectedFile.value.url, window.location.origin)
  url.searchParams.set('download', '1')
  return url.toString()
})
const effectiveFileKind = computed(() => selectedFileKind.value === 'html' ? detectedPreviewKind.value : selectedFileKind.value)
const snapshotURL = computed(() => selectedFile.value?.snapshotUrl || '')

watch([selectedFileURL, selectedFileKind], async ([url, kind]) => {
  htmlPreview.value = ''
  detectedPreviewKind.value = 'html'
  htmlPreviewError.value = false
  htmlPreviewLoading.value = kind === 'html' && Boolean(url)
  if (kind !== 'html' || !url) return
  try {
    const response = await fetch(url, { credentials: 'same-origin' })
    if (!response.ok) throw new Error(`policy html ${response.status}`)
    const bytes = await response.arrayBuffer()
    const header = new TextDecoder().decode(bytes.slice(0, 12))
    if (header.startsWith('%PDF-')) {
      detectedPreviewKind.value = 'pdf'
      return
    }
    let text = new TextDecoder('utf-8', { fatal: false }).decode(bytes)
    // A few examination-office pages are GBK/GB18030. Preserve Chinese text
    // instead of rendering replacement characters in the srcdoc iframe.
    if ((text.match(/�/g)?.length ?? 0) > 3) {
      text = new TextDecoder('gb18030', { fatal: false }).decode(bytes)
    }
    htmlPreview.value = text
  } catch {
    htmlPreviewError.value = true
  } finally {
    htmlPreviewLoading.value = false
  }
}, { immediate: true })

function goBack() {
  router.push(provinceName.value ? `/knowledge/${encodeURIComponent(provinceName.value)}` : '/knowledge')
}

</script>

<template>
  <main class="detail-page policy-document-page">
    <section v-if="document" class="policy-workspace">
      <header class="policy-document-toolbar">
        <button type="button" class="policy-toolbar-back" aria-label="返回省份资料包" title="返回省份资料包" @click="goBack">
          <ChevronLeft :size="19" />
        </button>
        <div class="policy-toolbar-title">
          <small>政策库 / {{ provinceName }}</small>
          <h1>{{ policyDisplayName(document) }}</h1>
          <div class="policy-toolbar-meta">
            <span class="verified"><CheckCircle2 :size="14" /> 已复核</span>
            <span>{{ document.dataYear }} 年</span>
            <span v-for="tag in documentTags.slice(0, 2)" :key="tag"># {{ tag }}</span>
          </div>
        </div>
        <div class="policy-toolbar-actions">
          <a :href="document.source.url || document.url" target="_blank" rel="noreferrer">
            官方来源 <ExternalLink :size="15" />
          </a>
          <a v-if="selectedFile" :href="downloadFileURL">
            <Download :size="15" /> 下载
          </a>
        </div>
      </header>

      <section class="policy-document-layout">
        <aside class="policy-document-toc">
          <div class="policy-toc-heading">
            <strong><BookOpenCheck :size="16" /> {{ provinceName }}资料</strong>
            <small>{{ documents.length }} 条记录</small>
          </div>
          <nav aria-label="政策记录">
            <details v-for="group in documentGroups" :key="group.group" :open="group.group === 'ordinary' || group.items.some((item) => item.id === document?.id)">
              <summary>{{ group.label }} <small>{{ group.items.length }}</small></summary>
              <RouterLink
                v-for="item in group.items"
                :key="item.id"
                :to="policyDocumentPath(provinceName, item.id)"
                :class="{ active: item.id === document.id }"
              >
                <small>{{ item.stage || item.type }}</small>
                <span>{{ policyDisplayName(item) }}</span>
              </RouterLink>
            </details>
          </nav>
          <div v-if="localFileGroups.length" class="policy-toc-files">
            <strong>本地文件</strong>
            <details v-for="group in localFileGroups" :key="group.group" :open="group.group === 'ordinary' || group.items.some((item) => item.name === selectedFile?.name)">
              <summary>{{ group.label }} <small>{{ group.items.length }}</small></summary>
              <RouterLink
                v-for="file in group.items"
                :key="file.url"
                :to="{ path: policyDocumentPath(provinceName, document.id), query: { file: file.name } }"
                :class="{ active: file.name === selectedFile?.name }"
              >
                <FileText :size="14" />
                <span>{{ policyDisplayName(file) }}</span>
                <small>{{ file.type }}</small>
              </RouterLink>
            </details>
          </div>
        </aside>

        <article class="policy-document-reader">
          <section v-if="selectedFile" class="policy-file-viewer">
            <div class="policy-file-viewer-head">
              <div>
                <strong>{{ selectedFileDisplayName }}</strong>
                <small>{{ selectedFile.stage || selectedFile.type }} · {{ selectedFile.year || document.dataYear }} · {{ Math.ceil(selectedFile.sizeBytes / 1024) }} KB</small>
              </div>
            </div>
            <div v-if="htmlPreviewLoading" class="policy-file-loading">
              <RefreshCcw class="state-spin" :size="22" /> 正在载入文件
            </div>
            <iframe v-else-if="effectiveFileKind === 'pdf'" :src="selectedFileURL" :title="`${selectedFileDisplayName}在线预览`" class="policy-file-frame" />
            <iframe v-else-if="selectedFileKind === 'office'" :src="officeViewerURL" :title="`${selectedFileDisplayName}在线预览`" class="policy-file-frame" />
            <iframe v-else-if="effectiveFileKind === 'html' && htmlPreview" :srcdoc="htmlPreview" :title="`${selectedFileDisplayName}在线预览`" class="policy-file-frame policy-html-frame" sandbox="allow-same-origin" />
            <div v-else-if="snapshotURL" class="policy-file-snapshot">
              <img :src="snapshotURL" :alt="`${selectedFileDisplayName}官方页面快照`" loading="lazy" />
              <p>官方页面快照 · {{ selectedFile.snapshotCapturedAt || '采集时间未记录' }}</p>
              <div><a :href="document.source.url || document.url" target="_blank" rel="noreferrer">打开官方来源</a><a :href="downloadFileURL">下载原文件</a></div>
            </div>
            <div v-else class="policy-file-unavailable">
              <FileText :size="22" />
              <p>{{ htmlPreviewError ? '网页内容载入失败，可打开官方来源或下载原文件。' : '该文件格式暂不支持网页内预览。' }}</p>
              <div>
                <a :href="document.source.url || document.url" target="_blank" rel="noreferrer">打开官方来源</a>
                <a :href="downloadFileURL">下载原文件</a>
              </div>
            </div>
          </section>

          <section v-else class="policy-file-empty">
            <FileText :size="28" />
            <h2>暂无本地文件</h2>
            <p>可以直接查看考试院官方来源。</p>
            <a :href="document.source.url || document.url" target="_blank" rel="noreferrer">打开官方来源 <ExternalLink :size="14" /></a>
          </section>

          <details class="policy-review-details">
            <summary>复核信息与使用说明</summary>
            <div>
              <p>{{ document.summary || '暂无摘要，请打开官方来源核对完整内容。' }}</p>
              <dl>
                <div><dt>来源</dt><dd>{{ document.source.name }}</dd></div>
                <div><dt>适用范围</dt><dd>{{ document.scope }}</dd></div>
                <div><dt>采集时间</dt><dd>{{ document.capturedAt }}</dd></div>
                <div><dt>方法说明</dt><dd>{{ document.methodology || '暂无方法说明。' }}</dd></div>
              </dl>
            </div>
          </details>
        </article>
      </section>
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
