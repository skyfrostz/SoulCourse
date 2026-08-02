<script setup lang="ts">
import { ChevronLeft, ChevronRight, RotateCcw, X, ZoomIn, ZoomOut } from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { appAssetUrl } from '../lib/runtime'

const props = defineProps<{
  images: string[]
  title: string
}>()

const currentIndex = ref(0)
const previewIndex = ref<number | null>(null)
const zoom = ref(1)
const failedImages = ref(new Set<string>())
const retryTokens = ref<Record<string, number>>({})
const touchStartX = ref<number | null>(null)

let previewScrollLockCount = 0
let previewScrollLockOverflow = ''

function lockPreviewScroll() {
  if (previewScrollLockCount === 0) previewScrollLockOverflow = document.body.style.overflow
  previewScrollLockCount += 1
  document.body.style.overflow = 'hidden'
}

function unlockPreviewScroll() {
  if (previewScrollLockCount === 0) return
  previewScrollLockCount -= 1
  if (previewScrollLockCount === 0) document.body.style.overflow = previewScrollLockOverflow
}

const total = computed(() => props.images.length)
const currentUrl = computed(() => props.images[currentIndex.value] ?? '')
const previewUrl = computed(() => previewIndex.value === null ? '' : props.images[previewIndex.value] ?? '')
const visibleDots = computed(() => total.value <= 9 ? props.images.map((_, index) => index) : [])

function normalizedIndex(index: number) {
  return total.value ? (index + total.value) % total.value : 0
}

function move(offset: number) {
  if (total.value < 2) return
  currentIndex.value = normalizedIndex(currentIndex.value + offset)
}

function openPreview() {
  if (!currentUrl.value) return
  previewIndex.value = currentIndex.value
  zoom.value = 1
}

function closePreview() {
  previewIndex.value = null
  zoom.value = 1
}

function movePreview(offset: number) {
  if (previewIndex.value === null || total.value < 2) return
  previewIndex.value = normalizedIndex(previewIndex.value + offset)
  currentIndex.value = previewIndex.value
  zoom.value = 1
}

function setZoom(nextZoom: number) {
  zoom.value = Math.min(4, Math.max(0.5, Number(nextZoom.toFixed(2))))
}

function markFailed(url: string) {
  failedImages.value = new Set(failedImages.value).add(url)
}

function retry(url: string) {
  failedImages.value = new Set([...failedImages.value].filter((item) => item !== url))
  retryTokens.value = { ...retryTokens.value, [url]: (retryTokens.value[url] ?? 0) + 1 }
}

function preload(index: number) {
  const url = props.images[normalizedIndex(index)]
  if (!url || failedImages.value.has(url)) return
  const image = new Image()
  image.src = appAssetUrl(url)
}

function handleTouchStart(event: TouchEvent) {
  touchStartX.value = event.changedTouches[0]?.clientX ?? null
}

function handleTouchEnd(event: TouchEvent) {
  if (touchStartX.value === null) return
  const distance = (event.changedTouches[0]?.clientX ?? touchStartX.value) - touchStartX.value
  touchStartX.value = null
  if (Math.abs(distance) >= 42) move(distance > 0 ? -1 : 1)
}

function handleKeydown(event: KeyboardEvent) {
  if (previewIndex.value === null) {
    if (event.key === 'ArrowLeft') move(-1)
    else if (event.key === 'ArrowRight') move(1)
    return
  }
  if (event.key === 'Escape') closePreview()
  else if (event.key === 'ArrowLeft') movePreview(-1)
  else if (event.key === 'ArrowRight') movePreview(1)
  else if (event.key === '+' || event.key === '=') setZoom(zoom.value + 0.2)
  else if (event.key === '-') setZoom(zoom.value - 0.2)
  else if (event.key === '0') setZoom(1)
}

watch([currentIndex, () => props.images], () => {
  preload(currentIndex.value - 1)
  preload(currentIndex.value + 1)
}, { immediate: true })

watch(() => props.images, (images) => {
  if (currentIndex.value >= images.length) currentIndex.value = 0
  closePreview()
})

watch(previewIndex, (index) => {
  if (index === null) unlockPreviewScroll()
  else lockPreviewScroll()
})

onMounted(() => window.addEventListener('keydown', handleKeydown))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
  if (previewIndex.value !== null) unlockPreviewScroll()
})
</script>

<template>
  <section class="post-image-carousel" :aria-label="`${title} 图片轮播`">
    <div
      class="carousel-stage"
      role="group"
      :aria-label="`第 ${currentIndex + 1} 张，共 ${total} 张`"
      @touchstart.passive="handleTouchStart"
      @touchend.passive="handleTouchEnd"
    >
      <button v-if="total > 1" class="carousel-arrow previous" type="button" aria-label="上一张图片" @click="move(-1)">
        <ChevronLeft :size="24" />
      </button>
      <button v-if="currentUrl && !failedImages.has(currentUrl)" class="carousel-image-button" type="button" aria-label="点击放大当前图片" @click="openPreview">
        <img
          :key="`${currentUrl}-${retryTokens[currentUrl] ?? 0}`"
          :src="appAssetUrl(currentUrl)"
          :alt="`${title} 第 ${currentIndex + 1} 张图片`"
          @error="markFailed(currentUrl)"
        />
      </button>
      <div v-else class="carousel-error">
        <strong>图片加载失败</strong>
        <button type="button" @click="retry(currentUrl)">重试</button>
      </div>
      <button v-if="total > 1" class="carousel-arrow next" type="button" aria-label="下一张图片" @click="move(1)">
        <ChevronRight :size="24" />
      </button>
      <span v-if="total > 1" class="carousel-counter" aria-live="polite" style="background-color: #111827; color: #ffffff">{{ currentIndex + 1 }} / {{ total }}</span>
    </div>
    <div v-if="total > 1 && total <= 9" class="carousel-dots" role="tablist" aria-label="选择图片">
      <button
        v-for="index in visibleDots"
        :key="index"
        type="button"
        role="tab"
        :aria-label="`查看第 ${index + 1} 张图片`"
        :aria-selected="index === currentIndex"
        :class="{ active: index === currentIndex }"
        @click="currentIndex = index"
      />
    </div>

    <Teleport to="body">
      <div v-if="previewIndex !== null && previewUrl" class="carousel-lightbox" role="dialog" aria-modal="true" aria-label="帖子图片预览" @click.self="closePreview">
        <div class="carousel-lightbox-topbar">
          <span>{{ previewIndex + 1 }} / {{ total }}</span>
          <div class="carousel-lightbox-tools">
            <button type="button" aria-label="缩小图片" title="缩小" @click="setZoom(zoom - 0.2)"><ZoomOut :size="20" /></button>
            <strong>{{ Math.round(zoom * 100) }}%</strong>
            <button type="button" aria-label="放大图片" title="放大" @click="setZoom(zoom + 0.2)"><ZoomIn :size="20" /></button>
            <button type="button" aria-label="重置缩放" title="重置缩放" @click="setZoom(1)"><RotateCcw :size="19" /></button>
            <button type="button" aria-label="关闭图片预览" title="关闭" @click="closePreview"><X :size="22" /></button>
          </div>
        </div>
        <button v-if="total > 1" class="carousel-lightbox-nav previous" type="button" aria-label="上一张" @click="movePreview(-1)"><ChevronLeft :size="30" /></button>
        <div class="carousel-lightbox-stage" @wheel.prevent="setZoom(zoom + ($event.deltaY < 0 ? 0.2 : -0.2))">
          <img :src="appAssetUrl(previewUrl)" :alt="`${title} 第 ${previewIndex + 1} 张图片`" :style="{ transform: `scale(${zoom})` }" />
        </div>
        <button v-if="total > 1" class="carousel-lightbox-nav next" type="button" aria-label="下一张" @click="movePreview(1)"><ChevronRight :size="30" /></button>
      </div>
    </Teleport>
  </section>
</template>

<style scoped>
.post-image-carousel { min-width: 0; }
.carousel-stage { position: relative; display: grid; min-height: 360px; height: min(78vh, 760px); place-items: center; overflow: hidden; background: #202124; }
.carousel-image-button { display: flex; width: 100%; height: 100%; min-width: 0; min-height: 0; align-items: center; justify-content: center; padding: 0; border: 0; background: transparent; }
.carousel-image-button img { display: block; flex: 0 1 auto; width: auto; height: auto; max-width: 100%; max-height: 100%; object-fit: contain; user-select: none; }
.carousel-arrow { position: absolute; z-index: 1; top: 50%; display: grid; width: 48px; height: 48px; place-items: center; padding: 0; border: 1px solid rgba(255,255,255,.24); border-radius: 50%; background: rgba(0,0,0,.46); color: #fff; transform: translateY(-50%); }
.carousel-arrow:hover { background: rgba(0,0,0,.7); }
.carousel-arrow.previous { left: 16px; }
.carousel-arrow.next { right: 16px; }
.carousel-counter { position: absolute; z-index: 2; right: 16px; bottom: 16px; padding: 6px 9px; border-radius: 5px; background: #111827 !important; color: #fff !important; font-size: 12px; font-weight: 800; isolation: isolate; }
.carousel-dots { display: flex; justify-content: center; gap: 7px; padding: 12px; background: #202124; }
.carousel-dots button { width: 8px; height: 8px; padding: 0; border: 0; border-radius: 50%; background: #777; }
.carousel-dots button.active { background: #fff; transform: scale(1.25); }
.carousel-error { display: grid; gap: 12px; place-items: center; color: #d1d5db; }
.carousel-error button { min-height: 40px; padding: 0 14px; border: 1px solid #6b7280; border-radius: 6px; background: #374151; color: #fff; }
.carousel-lightbox { position: fixed; inset: 0; z-index: 1000; display: grid; grid-template-columns: 72px minmax(0, 1fr) 72px; grid-template-rows: 64px minmax(0, 1fr); background: rgba(0,0,0,.94); color: #fff; }
.carousel-lightbox-topbar { grid-column: 1 / -1; display: flex; align-items: center; justify-content: space-between; padding: 0 18px; border-bottom: 1px solid rgba(255,255,255,.12); font-size: 13px; font-weight: 750; }
.carousel-lightbox-tools { display: flex; align-items: center; gap: 6px; }
.carousel-lightbox-tools button, .carousel-lightbox-nav { display: grid; width: 40px; height: 40px; place-items: center; padding: 0; border: 1px solid rgba(255,255,255,.18); border-radius: 7px; background: rgba(255,255,255,.08); color: #fff; }
.carousel-lightbox-tools strong { min-width: 48px; text-align: center; font-size: 12px; }
.carousel-lightbox-stage { display: grid; min-width: 0; min-height: 0; place-items: center; overflow: hidden; }
.carousel-lightbox-stage img { display: block; max-width: calc(100vw - 160px); max-height: calc(100vh - 88px); object-fit: contain; transform-origin: center; transition: transform 120ms ease; user-select: none; }
.carousel-lightbox-nav { align-self: center; justify-self: center; }
@media (max-width: 1023px) { .carousel-stage { height: clamp(360px, 68svh, 680px); } }
@media (max-width: 760px) { .carousel-arrow { width: 44px; height: 44px; } .carousel-arrow.previous { left: 10px; } .carousel-arrow.next { right: 10px; } .carousel-lightbox { grid-template-columns: 52px minmax(0, 1fr) 52px; grid-template-rows: auto minmax(0, 1fr); } .carousel-lightbox-topbar { min-height: 56px; padding: 8px 10px; } .carousel-lightbox-tools { gap: 3px; } .carousel-lightbox-tools button, .carousel-lightbox-nav { width: 36px; height: 36px; } .carousel-lightbox-tools strong, .carousel-lightbox-tools button:nth-of-type(3) { display: none; } .carousel-lightbox-stage img { max-width: calc(100vw - 108px); max-height: calc(100vh - 76px); } }
@media (prefers-reduced-motion: reduce) { .carousel-lightbox-stage img { transition: none; } }
</style>
