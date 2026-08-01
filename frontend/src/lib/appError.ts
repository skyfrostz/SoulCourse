import { ref } from 'vue'

export type AppErrorKind = 'chunk' | 'runtime'

export interface AppErrorState {
  kind: AppErrorKind
  title: string
  message: string
}

export const appError = ref<AppErrorState | null>(null)

export function showAppError(error: unknown, fallbackKind: AppErrorKind = 'runtime') {
  const kind = isChunkLoadError(error) ? 'chunk' : fallbackKind
  appError.value = {
    kind,
    title: kind === 'chunk' ? '页面资源加载失败' : '页面暂时无法显示',
    message: kind === 'chunk'
      ? '网络中断或版本刚刚发布，刷新后通常可以恢复。'
      : '页面运行时遇到问题，可以刷新或返回首页继续使用。',
  }
}

export function clearAppError() {
  appError.value = null
}

export function isChunkLoadError(error: unknown) {
  const message = error instanceof Error ? error.message : String(error)
  return /loading chunk|chunkloaderror|failed to fetch dynamically imported module|importing a module script failed/i.test(message)
}
