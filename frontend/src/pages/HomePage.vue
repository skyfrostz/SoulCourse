<script setup lang="ts">
import FeedGrid from '../components/FeedGrid.vue'
import FilterRail from '../components/FilterRail.vue'
import InsightPanel from '../components/InsightPanel.vue'
import PostDetailModal from '../components/PostDetailModal.vue'
import { useForumData } from '../composables/useForumData'
import { useGlobalSearch } from '../composables/useGlobalSearch'
import { useOnlineState } from '../composables/useOnlineState'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const { posts, insights, topics, isLoading, hasError, hasMore, refetch } = useForumData()
const { isOffline } = useOnlineState()
const railCollapsed = ref(false)
const route = useRoute()
const router = useRouter()
const { syncSearchFromRoute } = useGlobalSearch()
const postId = computed(() => {
  const value = Number(route.query.post)
  return Number.isInteger(value) && value > 0 ? value : null
})
const targetSection = computed(() => route.query.targetSection === 'comments' ? 'comments' : undefined)

function closePostDetail() {
  const query = { ...route.query }
  delete query.post
  delete query.targetSection
  void router.replace({ name: 'home', query })
}

watch(
  () => route.query.q,
  () => syncSearchFromRoute(),
  { immediate: true },
)
</script>

<template>
  <main class="workspace forum-home-workspace" :class="{ 'filter-collapsed': railCollapsed }">
    <FilterRail :collapsed="railCollapsed" @toggle-collapse="railCollapsed = !railCollapsed" />
    <FeedGrid :posts="posts" :is-loading="isLoading" :has-more="hasMore" :has-error="hasError" :is-offline="isOffline" @retry="refetch" />
    <InsightPanel :insights="insights" :topics="topics" />
    <PostDetailModal v-if="postId" :post-id="postId" :target-section="targetSection" @close="closePostDetail" />
  </main>
</template>
