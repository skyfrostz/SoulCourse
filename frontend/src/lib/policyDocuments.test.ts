import { describe, expect, it } from 'vitest'
import {
  getPolicyExamGroup,
  policyDisplayName,
  sortPolicyRecords,
} from './api'

describe('policy document presentation metadata', () => {
  it('puts ordinary gaokao records before supplemental exam types', () => {
    const sorted = sortPolicyRecords([
      { title: '成人高考报名办法', type: '招生政策' },
      { title: '普通高校招生志愿填报办法', type: '招生政策', stage: '志愿' },
      { title: '体育类专业招生办法', type: '招生政策' },
      { title: '普通高校招生报名通知', type: '招生政策', stage: '报名' },
    ])

    expect(sorted.map((item) => getPolicyExamGroup(item))).toEqual(['ordinary', 'ordinary', 'sports', 'adult'])
    expect(sorted[0].title).toContain('报名')
  })

  it('prefers an explicit display name and falls back to legacy names', () => {
    expect(policyDisplayName({ displayName: '2026年普通高考报名通知', name: '0001.pdf' })).toBe('2026年普通高考报名通知')
    expect(policyDisplayName({ name: '0001.pdf' })).toBe('0001.pdf')
  })
})
