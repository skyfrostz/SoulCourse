<script setup lang="ts">
import { computed, onErrorCaptured, onMounted, onUnmounted, ref, watch } from 'vue'
import { App as CapacitorApp } from '@capacitor/app'
import AuthModal from './components/AuthModal.vue'
import MobileBottomNav from './components/MobileBottomNav.vue'
import PublishModal from './components/PublishModal.vue'
import TopNav from './components/TopNav.vue'
import MobileOnboarding from './components/MobileOnboarding.vue'
import { useForumData } from './composables/useForumData'
import { appError, clearAppError, showAppError } from './lib/appError'
import { useForumStore } from './stores/forum'
import { useRoute, useRouter } from 'vue-router'
import { isNativeApp } from './lib/mobile'
import { installMobileLifecycle } from './lib/mobileLifecycle'
import { Preferences } from '@capacitor/preferences'

const forumStore = useForumStore()
const { source } = useForumData()
const route = useRoute()
const router = useRouter()
const isAdminLayout = computed(() => route.meta.layout === 'admin')
const isImmersiveLayout = computed(() => route.meta.layout === 'immersive')
const viewKey = computed(() => route.name === 'home' ? 'home' : route.fullPath)
const showOnboarding = ref(false)
let removeLifecycle: (() => void) | undefined

onMounted(async () => {
  if (isNativeApp) {
    const { value } = await Preferences.get({ key: 'mobile_onboarding_v1_complete' })
    showOnboarding.value = value !== 'true'
    removeLifecycle = installMobileLifecycle(router)
    await CapacitorApp.addListener('backButton', ({ canGoBack }) => {
      if (forumStore.authOpen) { forumStore.authOpen = false; return }
      if (forumStore.publishOpen) { forumStore.publishOpen = false; return }
      if (canGoBack) void router.back()
      else void CapacitorApp.minimizeApp()
    })
  }
  if (forumStore.isAuthed) void forumStore.hydrateAccount()
})

onUnmounted(() => removeLifecycle?.())

async function finishOnboarding(profile: { province: string; grade: string; role: string }) {
  await Preferences.set({ key: 'mobile_onboarding_v1_complete', value: 'true' })
  await Preferences.set({ key: 'mobile_onboarding_profile', value: JSON.stringify(profile) })
  showOnboarding.value = false
}

watch(() => route.fullPath, () => clearAppError())

onErrorCaptured((error) => {
  showAppError(error, 'runtime')
  return false
})

function reloadPage() {
  window.location.reload()
}
</script>

<template>
  <div class="app-shell" :class="{ 'is-home-route': route.path === '/', 'is-admin-route': isAdminLayout, 'is-immersive-route': isImmersiveLayout }">
    <TopNav v-if="!isAdminLayout && !isImmersiveLayout" :source="source" />
    <main v-if="appError" class="detail-page app-error-page">
      <section class="empty-state detail-empty-state public-page-state">
        <h1>{{ appError.title }}</h1>
        <p>{{ appError.message }}</p>
        <div class="state-action-row">
          <button class="primary-wide compact" type="button" @click="reloadPage">刷新页面</button>
          <RouterLink class="ghost-button compact" to="/">返回首页</RouterLink>
        </div>
      </section>
    </main>
    <RouterView v-else v-slot="{ Component }">
      <Transition name="page-flow" mode="out-in">
        <component :is="Component" :key="viewKey" />
      </Transition>
    </RouterView>
    <MobileBottomNav v-if="!isAdminLayout && !isImmersiveLayout" />
    <AuthModal v-if="!isAdminLayout && !isImmersiveLayout && forumStore.authOpen" />
    <div v-if="!isAdminLayout && !isImmersiveLayout && forumStore.publishOpen" v-show="!forumStore.authOpen">
      <PublishModal />
    </div>
    <MobileOnboarding v-if="showOnboarding" @complete="finishOnboarding" />
  </div>
</template>
