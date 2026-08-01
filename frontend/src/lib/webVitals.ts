import { onCLS, onINP, onLCP, type Metric } from 'web-vitals'
import { defaultApiBasePath } from './runtime'

export interface WebVitalPayload {
  name: 'LCP' | 'INP' | 'CLS'
  value: number
  rating: 'good' | 'needs-improvement' | 'poor'
}

export function toWebVitalPayload(metric: Metric): WebVitalPayload | null {
  if (!['LCP', 'INP', 'CLS'].includes(metric.name) || !Number.isFinite(metric.value) || metric.value < 0) return null
  return { name: metric.name as WebVitalPayload['name'], value: metric.value, rating: metric.rating }
}

export function reportWebVitals() {
  if (navigator.webdriver) return
  const endpoint = `${defaultApiBasePath()}/telemetry/web-vitals`
  const report = (metric: Metric) => {
    const payload = toWebVitalPayload(metric)
    if (!payload) return
    const body = JSON.stringify(payload)
    if (navigator.sendBeacon?.(endpoint, new Blob([body], { type: 'application/json' }))) return
    void fetch(endpoint, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body, keepalive: true }).catch(() => {})
  }
  onLCP(report)
  onINP(report)
  onCLS(report)
}
