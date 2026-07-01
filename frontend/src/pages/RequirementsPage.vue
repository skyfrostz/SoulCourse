<script setup lang="ts">
import { computed, ref } from 'vue'
import { ChevronLeft, Search, ShieldCheck } from '@lucide/vue'
import { useRouter } from 'vue-router'
import { majorRequirements } from '../lib/majorRequirements'

const router = useRouter()
const keyword = ref('')
const results = computed(() => {
  const q = keyword.value.trim()
  if (!q) return majorRequirements
  return majorRequirements.filter((item) =>
    [item.major, item.category, item.suggestedCombination, item.requiredSubjects.join('')]
      .some((value) => value.includes(q)),
  )
})
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="requirements-hero">
      <div>
        <div class="breadcrumb">工具 / 选科要求查询</div>
        <h1>高校专业选科要求查询</h1>
        <p>输入专业名称，快速查看常见选科要求、推荐组合和风险提示。MVP 阶段先提供高频专业库，后续可扩展为完整院校专业数据库。</p>
      </div>
      <label class="requirement-search">
        <Search :size="18" />
        <input v-model="keyword" placeholder="搜索：临床医学、计算机、法学、师范..." />
      </label>
    </section>

    <section class="requirement-results">
      <article v-for="item in results" :key="item.major" class="requirement-card">
        <small>{{ item.category }}</small>
        <h2>{{ item.major }}</h2>
        <div class="requirement-subjects">
          <span v-for="subject in item.requiredSubjects" :key="subject">{{ subject }}</span>
        </div>
        <p><strong>建议组合：</strong>{{ item.suggestedCombination }}</p>
        <p><strong>风险提示：</strong>{{ item.risk }}</p>
        <footer><ShieldCheck :size="16" /> 来源：{{ item.source }}</footer>
      </article>
    </section>
  </main>
</template>
