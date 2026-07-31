import { describe, expect, it } from 'vitest'
import { difficultyFromTags, problemDifficultyOptions } from '../src/features/problems/problemMeta'

describe('problem difficulty vocabulary', () => {
  it('uses the six canonical values in display order', () => {
    expect(problemDifficultyOptions).toEqual(['入门', '基础', '普及', '提高', '综合', '挑战'])
  })

  it('keeps challenge distinct from comprehensive', () => {
    expect(difficultyFromTags({ labels: ['挑战'] })).toBe('挑战')
  })
})
