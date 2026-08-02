import { App } from '@capacitor/app'
import { isNativeApp } from './mobile'
import { refreshMobileSession } from './api'
import { useForumStore } from '../stores/forum'
import type { Router } from 'vue-router'

export function installMobileLifecycle(router: Router) {
  if (!isNativeApp) return () => undefined
  const forumStore = useForumStore()
  const listener = App.addListener('appStateChange', async ({ isActive }) => {
    if (!isActive || !forumStore.session?.expiresAt) return
    const expiresAt = new Date(forumStore.session.expiresAt).getTime()
    if (expiresAt - Date.now() > 7 * 24 * 60 * 60 * 1000) {
      void forumStore.hydrateAccount()
      return
    }
    try {
      forumStore.setSession(await refreshMobileSession())
    } catch {
      forumStore.handleUnauthorized()
    }
  })
  const urlListener = App.addListener('appUrlOpen', ({ url }) => {
    try {
      const parsed = new URL(url)
      if (parsed.host === 'soulcourse.cn' || parsed.protocol === 'soulcourse:') {
        const target = `${parsed.pathname}${parsed.search}${parsed.hash}`
        if (target && !target.startsWith('/welcome') && !target.startsWith('/admin')) void router.push(target)
      }
    } catch {
      // Ignore malformed external intents.
    }
  })
  return () => {
    void listener.then((handle) => handle.remove())
    void urlListener.then((handle) => handle.remove())
  }
}
