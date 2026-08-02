<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { X } from '@lucide/vue'
import PostDetailPage from '../pages/PostDetailPage.vue'

const props = defineProps<{ postId: number; targetSection?: string }>()
const emit = defineEmits<{ close: [] }>()
const dialog = ref<HTMLElement | null>(null)
const previousActiveElement = ref<HTMLElement | null>(null)
const previousOverflow = ref('')
const previousScrollY = ref(0)

function close() {
  emit('close')
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    close()
    return
  }
  if (event.key !== 'Tab' || !dialog.value) return
  const focusable = [...dialog.value.querySelectorAll<HTMLElement>('button, a[href], input, textarea, select, [tabindex]:not([tabindex="-1"])')]
    .filter((element) => !element.hasAttribute('disabled') && element.offsetParent !== null)
  if (!focusable.length) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

onMounted(async () => {
  previousActiveElement.value = document.activeElement instanceof HTMLElement ? document.activeElement : null
  previousOverflow.value = document.body.style.overflow
  previousScrollY.value = window.scrollY
  document.body.style.overflow = 'hidden'
  window.addEventListener('keydown', handleKeydown)
  await nextTick()
  dialog.value?.querySelector<HTMLElement>('[aria-label="关闭帖子详情"]')?.focus()
  if (props.targetSection === 'comments') {
    window.setTimeout(() => document.getElementById('post-comments')?.scrollIntoView({ block: 'start' }), 150)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
  document.body.style.overflow = previousOverflow.value
  window.scrollTo({ top: previousScrollY.value, behavior: 'auto' })
  previousActiveElement.value?.focus()
})
</script>

<template>
  <Teleport to="body">
    <div class="post-detail-modal-backdrop" @click.self="close">
      <section ref="dialog" class="post-detail-modal" role="dialog" aria-modal="true" aria-label="帖子详情">
        <button class="post-detail-modal-close" type="button" aria-label="关闭帖子详情" @click="close">
          <X :size="22" />
        </button>
        <PostDetailPage :post-id="props.postId" mode="modal" />
      </section>
    </div>
  </Teleport>
</template>
