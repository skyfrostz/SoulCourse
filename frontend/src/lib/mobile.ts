import { Capacitor, registerPlugin } from '@capacitor/core'

export interface SecureSessionPlugin {
  get(): Promise<{ value?: string }>
  set(options: { value: string }): Promise<void>
  remove(): Promise<void>
}

export interface MobileAuthSession {
  user: import('../types/forum').User
  accessToken: string
  expiresAt: string
}

const secureSession = registerPlugin<SecureSessionPlugin>('SecureSession')
let accessToken = ''

export const isAndroidApp = Capacitor.getPlatform() === 'android'
export const isNativeApp = Capacitor.isNativePlatform()

export async function hydrateMobileSession() {
  if (!isNativeApp) return
  try {
    accessToken = (await secureSession.get()).value?.trim() ?? ''
  } catch {
    accessToken = ''
  }
}

export function mobileAccessToken() {
  return accessToken
}

export async function storeMobileAccessToken(token: string) {
  accessToken = token.trim()
  if (!isNativeApp) return
  await secureSession.set({ value: accessToken })
}

export async function clearMobileAccessToken() {
  accessToken = ''
  if (!isNativeApp) return
  try {
    await secureSession.remove()
  } catch {
    // The local session is cleared even if the native keystore is unavailable.
  }
}
