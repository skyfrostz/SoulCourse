<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ChevronLeft, ExternalLink, FileText, ShieldCheck } from '@lucide/vue'
import PostCard from '../components/PostCard.vue'
import { provinceKnowledge } from '../lib/knowledgeBase'
import { createProvincePolicyDocuments, policyDocumentPath } from '../lib/policyDocuments'
import { requirementData } from '../lib/realData'
import { samplePosts } from '../lib/sampleData'

const route = useRoute()
const router = useRouter()
const provinceName = computed(() => decodeURIComponent(String(route.params.province ?? '')))
const province = computed(() => provinceKnowledge.find((item) => item.province === provinceName.value))
const requirement = computed(() => requirementData.find((item) => item.province === provinceName.value))
const provincePosts = computed(() => {
  if (!province.value) return []
  const focused = samplePosts.filter((post) => post.province === province.value?.province)
  const byPolicy = samplePosts.filter(
    (post) =>
      post.province === '全国' ||
      post.tags.some((tag) => province.value?.focus.some((focus) => tag.includes(focus) || focus.includes(tag))),
  )
  const merged = new Map([...focused, ...byPolicy].map((post) => [post.id, post]))
  return Array.from(merged.values()).slice(0, 12)
})

const fileCards = computed(() => (province.value ? createProvincePolicyDocuments(province.value) : []))
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/knowledge')"><ChevronLeft :size="17" /> 返回政策库</button>

    <section v-if="province" class="province-detail-hero">
      <div>
        <div class="breadcrumb">政策库 / {{ province.region }} / {{ province.province }}</div>
        <h1>{{ province.province }}招生考试与选科文件</h1>
        <p>{{ province.status }}</p>
        <div class="overview-metrics">
          <span><ShieldCheck :size="18" /> {{ province.reformMode }}</span>
          <span v-for="tag in province.focus" :key="tag"># {{ tag }}</span>
        </div>
      </div>
      <a :href="province.portalUrl" target="_blank" rel="noreferrer" class="primary-wide compact">
        官方入口 <ExternalLink :size="15" />
      </a>
    </section>

    <section v-if="province" class="province-file-grid">
      <article v-for="file in fileCards" :key="file.title" class="province-file-card">
        <small><FileText :size="15" /> {{ file.type }}</small>
        <h2>{{ file.title }}</h2>
        <p>{{ file.abstract }}</p>
        <div class="province-file-actions">
          <RouterLink :to="policyDocumentPath(province.province, file.id)">
            <FileText :size="15" /> 查看网页化全文
          </RouterLink>
          <a :href="file.downloadUrl" target="_blank" rel="noreferrer">
            官方/下载入口 <ExternalLink :size="14" />
          </a>
        </div>
      </article>
    </section>

    <section v-if="province" class="province-content-grid">
      <article>
        <h2>文件内容速读</h2>
        <ul>
          <li v-for="item in province.checklist" :key="item">{{ item }}</li>
          <li v-if="requirement">{{ requirement.note }}</li>
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

    <section v-if="province" class="feed-panel province-post-panel">
      <div class="feed-toolbar">
        <div class="feed-tabs">
          <button class="active">{{ province.province }}相关笔记</button>
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
      <p>请返回政策库，从省份卡片进入。</p>
    </section>
  </main>
</template>
