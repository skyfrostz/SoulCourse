import { Preferences } from '@capacitor/preferences'
import { Device } from '@capacitor/device'
import { isNativeApp } from './mobile'
import { reportMobileTelemetry } from './api'

const consentKey = 'mobile_diagnostics_consent'
let enabled = false
let deviceInfo: { osVersion?: string; webViewVersion?: string } = {}

function androidApiLevel(version?: string) {
  const major = Number.parseInt(version?.split('.').at(0) || '', 10)
  return major >= 8 ? ({ 8: 26, 9: 28, 10: 29, 11: 30, 12: 31, 13: 33, 14: 34, 15: 35, 16: 36 }[major] || 36) : 26
}

function routeTemplate() {
  return `${window.location.pathname}${window.location.search}`
    .replace(/\/\d+(?=\/|$)/g, '/:id')
    .slice(0, 120)
}

export async function loadMobileTelemetryConsent() {
  if (!isNativeApp) return
  const [{ value }, info] = await Promise.all([
    Preferences.get({ key: consentKey }),
    Device.getInfo(),
  ])
  enabled = value === 'true'
  deviceInfo = { osVersion: info.osVersion, webViewVersion: info.webViewVersion }
}

export async function setMobileTelemetryConsent(value: boolean) {
  enabled = value
  await Preferences.set({ key: consentKey, value: String(value) })
}

export function installMobileTelemetry() {
  if (!isNativeApp) return () => undefined
  const report = (event: 'js_error' | 'native_error') => {
    if (!enabled) return
    void reportMobileTelemetry({
      event,
      appVersion: import.meta.env.VITE_APP_VERSION || 'dev',
      androidApi: androidApiLevel(deviceInfo.osVersion),
      webView: deviceInfo.webViewVersion || 'android-webview',
      route: routeTemplate(),
    }).catch(() => undefined)
  }
  const onError = () => report('js_error')
  const onRejection = () => report('js_error')
  window.addEventListener('error', onError)
  window.addEventListener('unhandledrejection', onRejection)
  return () => {
    window.removeEventListener('error', onError)
    window.removeEventListener('unhandledrejection', onRejection)
  }
}
