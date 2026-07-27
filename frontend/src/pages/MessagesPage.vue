<script setup lang="ts">
import { ChevronLeft, MessageCircle, PenLine } from '@lucide/vue'
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="messages-empty-page">
      <div class="messages-empty-icon"><MessageCircle :size="34" /></div>
      <span>私信</span>
      <h1>还没有会话</h1>
      <p>当真实用户之间建立沟通后，会话会显示在这里。我们不会用演示数据填充你的消息列表。</p>
      <button type="button" @click="router.push('/')"><PenLine :size="16" /> 先去参与讨论</button>
    </section>
  </main>
</template>
