import { onBeforeUnmount, onMounted, ref } from 'vue'

export function useOnlineState() {
  const isOffline = ref(typeof navigator !== 'undefined' ? !navigator.onLine : false)

  function sync() {
    isOffline.value = typeof navigator !== 'undefined' ? !navigator.onLine : false
  }

  onMounted(() => {
    window.addEventListener('online', sync)
    window.addEventListener('offline', sync)
  })

  onBeforeUnmount(() => {
    window.removeEventListener('online', sync)
    window.removeEventListener('offline', sync)
  })

  return { isOffline }
}
