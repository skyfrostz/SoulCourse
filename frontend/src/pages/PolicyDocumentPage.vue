<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { BookOpenCheck, ChevronLeft, Download, ExternalLink, FileText, Search, ShieldCheck } from '@lucide/vue'
import { provinceKnowledge } from '../lib/knowledgeBase'
import { createProvincePolicyDocuments, findProvincePolicyDocument, policyDocumentPath } from '../lib/policyDocuments'
import { useForumStore } from '../stores/forum'

const route = useRoute()
const router = useRouter()
const forumStore = useForumStore()
const provinceName = computed(() => decodeURIComponent(String(route.params.province ?? '')))
const documentId = computed(() => String(route.params.documentId ?? ''))
const province = computed(() => provinceKnowledge.find((item) => item.province === provinceName.value))
const documents = computed(() => (province.value ? createProvincePolicyDocuments(province.value) : []))
const document = computed(() => findProvincePolicyDocument(province.value, documentId.value))

function goBack() {
  router.push(province.value ? `/knowledge/${encodeURIComponent(province.value.province)}` : '/knowledge')
}

function searchInForum(query: string) {
  forumStore.setKeyword(query)
  router.push('/')
}
</script>

<template>
  <main class="detail-page policy-document-page">
    <button class="back-link" @click="goBack"><ChevronLeft :size="17" /> 返回省份资料包</button>

    <section v-if="province && document" class="policy-document-hero">
      <div>
        <div class="breadcrumb">政策库 / {{ province.province }} / PDF网页化阅读</div>
        <h1>{{ document.title }}</h1>
        <p>{{ document.subtitle }}。{{ document.abstract }}</p>
        <div class="overview-metrics">
          <span><ShieldCheck :size="18" /> {{ document.sourceName }}</span>
          <span v-for="tag in document.tags.slice(0, 4)" :key="tag"># {{ tag }}</span>
        </div>
      </div>
      <div class="policy-document-actions">
        <a :href="document.downloadUrl" target="_blank" rel="noreferrer" class="primary-wide compact">
          下载/检索 PDF <Download :size="15" />
        </a>
        <a :href="document.sourceUrl" target="_blank" rel="noreferrer">
          官方来源 <ExternalLink :size="15" />
        </a>
      </div>
    </section>

    <section v-if="province && document" class="policy-document-layout">
      <aside class="policy-document-toc">
        <strong><BookOpenCheck :size="17" /> {{ province.province }}文件目录</strong>
        <RouterLink
          v-for="item in documents"
          :key="item.id"
          :to="policyDocumentPath(province.province, item.id)"
          :class="{ active: item.id === document.id }"
        >
          <small>{{ item.type }}</small>
          <span>{{ item.title }}</span>
        </RouterLink>
      </aside>

      <article class="policy-document-reader">
        <section class="policy-reader-note">
          <FileText :size="20" />
          <div>
            <strong>网页化正文</strong>
            <p>
              本页将政策 PDF/附件的核对框架转换为站内可读内容，方便搜索、收藏和转发讨论。最终报考仍以官方 PDF、考试院公告和高校招生章程为准。
            </p>
          </div>
        </section>

        <section v-for="section in document.sections" :key="section.heading" class="policy-reader-section">
          <h2>{{ section.heading }}</h2>
          <p v-for="paragraph in section.paragraphs" :key="paragraph">{{ paragraph }}</p>
          <ul>
            <li v-for="bullet in section.bullets" :key="bullet">{{ bullet }}</li>
          </ul>
        </section>

        <section class="policy-check-board">
          <div>
            <h2>下载前核对清单</h2>
            <ol>
              <li v-for="item in document.checkItems" :key="item">{{ item }}</li>
            </ol>
          </div>
          <div>
            <h2>站内继续检索</h2>
            <button
              v-for="query in document.relatedQueries"
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

    <section v-else class="empty-state">
      <FileText :size="30" />
      <h2>没有找到这份政策文档</h2>
      <p>请返回政策库，从省份资料包重新进入。</p>
      <button class="primary-wide compact" type="button" @click="router.push('/knowledge')">返回政策库</button>
    </section>
  </main>
</template>
