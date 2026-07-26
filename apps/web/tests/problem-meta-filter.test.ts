import { describe, expect, it } from 'vitest'
import { problemMatchesFilters, type ProblemFilters } from '../src/features/problems/problemMeta'

const problem = {
  id: 7,
  display_code: 'T007',
  title: '两数之和',
  statement: '',
  tags: { labels: ['数组'] },
  difficulty: '基础',
  time_limit_ms: 1000,
  memory_limit_mb: 256,
  output_limit_kb: 1024,
  progress_status: 'accepted' as const
}

function filters(overrides: Partial<ProblemFilters> = {}): ProblemFilters {
  return {
    keyword: '',
    tag: '',
    difficulty: '',
    status: 'all',
    ...overrides
  }
}

describe('problem list filters', () => {
  it('matches display code, tag, difficulty, and status', () => {
    expect(problemMatchesFilters(problem, filters({ keyword: 'T007' }))).toBe(true)
    expect(problemMatchesFilters(problem, filters({ tag: '数组' }))).toBe(true)
    expect(problemMatchesFilters(problem, filters({ difficulty: '基础' }))).toBe(true)
    expect(problemMatchesFilters(problem, filters({ status: 'accepted' }))).toBe(true)
  })

  it('rejects a different difficulty', () => {
    expect(problemMatchesFilters(problem, filters({ difficulty: '提高' }))).toBe(false)
  })
})
