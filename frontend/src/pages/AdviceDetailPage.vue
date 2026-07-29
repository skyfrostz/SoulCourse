<script setup lang="ts">
import { ChevronLeft } from '@lucide/vue'
import { computed, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const postId = computed(() => Number(route.params.id))

watchEffect(() => {
  if (Number.isSafeInteger(postId.value) && postId.value > 0) {
    router.replace({ name: 'post-detail', params: { id: postId.value }, hash: route.hash })
  }
})
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/advice')"><ChevronLeft :size="17" /> 返回选科建议</button>
    <section class="empty-state detail-empty-state">
      <h1>建议内容已经迁入统一帖子库</h1>
      <p>旧的静态建议链接不再提供本地 mock 内容，请从建议页选择一条真实帖子。</p>
      <button class="primary-wide compact" @click="router.push('/advice')">浏览真实建议帖子</button>
    </section>
  </main>
</template>
