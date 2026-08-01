import { describe, expect, it } from 'vitest'
import { toWebVitalPayload } from './webVitals'

describe('web vitals telemetry', () => {
  it('keeps only anonymous metric fields', () => {
    const payload = toWebVitalPayload({ name: 'LCP', value: 1234, rating: 'good', delta: 1234, id: 'metric-id', entries: [], navigationType: 'navigate', navigationId: 1 })
    expect(payload).toEqual({ name: 'LCP', value: 1234, rating: 'good' })
    expect(payload).not.toHaveProperty('id')
  })
})
