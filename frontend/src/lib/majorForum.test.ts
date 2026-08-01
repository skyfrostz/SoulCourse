import { describe, expect, it } from 'vitest'
import { toMajorRequirementCard } from './majorForum'

describe('major requirement API mapping', () => {
  it('maps verified records without relying on local seed data', () => {
    const card = toMajorRequirementCard({
      id: 'req-1',
      title: '计算机科学与技术',
      type: '物理 + 化学',
      scope: '广东',
      coverageStatus: 'verified',
      dataYear: 2026,
      capturedAt: '2026-07-31T00:00:00Z',
      source: { name: '广东省教育考试院', url: 'https://example.test/source' },
      fileHash: 'sha256:test',
      methodology: '按官方目录逐校核验',
      summary: '理工专业要求摘要',
      tags: ['计算机', '广东'],
      url: 'https://example.test/record',
      requiredSubjects: ['物理', '化学'],
    })

    expect(card.major).toBe('计算机科学与技术')
    expect(card.requiredSubjects).toEqual(expect.arrayContaining(['物理', '化学']))
    expect(card.coverageStatus).toBe('verified')
    expect(card.sourceUrl).toBe('https://example.test/source')
  })

  it('does not infer required subjects from summaries or tags', () => {
    const card = toMajorRequirementCard({
      id: 'req-2', title: '示例专业', type: '专业要求', scope: '广东',
      coverageStatus: 'verified', dataYear: 2026, capturedAt: '2026-07-31T00:00:00Z',
      source: { name: '官方来源', url: 'https://example.test/source' },
      fileHash: 'sha256:test', methodology: '逐项复核',
      summary: '说明文字提到了物理和化学，但没有结构化必选科目。',
      tags: ['物理'], url: 'https://example.test/record',
    })

    expect(card.requiredSubjects).toEqual(['以官方目录为准'])
    expect(card.suggestedCombination).toBe('按官方目录逐校核对')
  })
})
