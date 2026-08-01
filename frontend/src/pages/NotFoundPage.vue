<script setup lang="ts">
import { ArrowLeft, Home, SearchX } from '@lucide/vue'
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <main class="not-found-page" aria-labelledby="not-found-title">
    <section class="not-found-panel">
      <SearchX :size="44" aria-hidden="true" />
      <p class="eyebrow">404</p>
      <h1 id="not-found-title">页面不存在</h1>
      <p>这个地址没有可用页面，可能是链接已变更或内容尚未开放。</p>
      <div class="not-found-actions">
        <button type="button" class="secondary-action" @click="router.back()">
          <ArrowLeft :size="18" aria-hidden="true" />
          返回上一页
        </button>
        <RouterLink class="primary-action" to="/">
          <Home :size="18" aria-hidden="true" />
          回到首页
        </RouterLink>
      </div>
    </section>
  </main>
</template>
