import type { Problem } from '../../api/client'

export type ProblemStatusFilter = 'all' | 'unattempted' | 'attempted' | 'accepted'

export interface ProblemFilters {
  keyword: string
  tags: string[]
  difficulty: string
  status: ProblemStatusFilter
}

export const problemStatusOptions: Array<{ label: string; value: ProblemStatusFilter }> = [
  { label: '全部状态', value: 'all' },
  { label: '未尝试', value: 'unattempted' },
  { label: '未通过', value: 'attempted' },
  { label: '已通过', value: 'accepted' }
]

export const problemDifficultyOptions = ['入门', '基础', '普及', '提高', '综合', '挑战'] as const

export function tagList(tags: unknown) {
  if (!tags) return []
  if (Array.isArray(tags)) return tags.map(String)
  if (typeof tags === 'object' && tags !== null) {
    const value = tags as { labels?: unknown[]; items?: unknown[] }
    if (Array.isArray(value.labels)) return value.labels.map(String)
    if (Array.isArray(value.items)) return value.items.map(String)
  }
  return []
}

export function progressLabel(status?: string) {
  if (status === 'accepted') return '通过'
  if (status === 'attempted') return '未通过'
  return '未尝试'
}

export function progressTag(status?: string): 'success' | 'warning' | 'info' {
  if (status === 'accepted') return 'success'
  if (status === 'attempted') return 'warning'
  return 'info'
}

export function problemDisplayCode(problem: Pick<Problem, 'id' | 'display_code'>) {
  return problem.display_code || '未编号'
}

export function problemLimitText(problem: Pick<Problem, 'time_limit_ms' | 'memory_limit_mb' | 'output_limit_kb'>) {
  return `${problem.time_limit_ms} ms / ${problem.memory_limit_mb} MB / ${problem.output_limit_kb} KB`
}

export function problemLimitLines(problem: Pick<Problem, 'time_limit_ms' | 'memory_limit_mb'>) {
  return [`时间限制：${problem.time_limit_ms} ms`, `内存限制：${problem.memory_limit_mb} MB`]
}

export function difficultyFromTags(tags: unknown) {
  const normalized = tagList(tags)
  const direct = normalized.find((tag) => problemDifficultyOptions.includes(tag as typeof problemDifficultyOptions[number]))
  if (direct) return direct
  const legacy = normalized.find((tag) => ['简单', '中等', '困难', '挑战', 'Easy', 'Medium', 'Hard'].includes(tag))
  if (['简单', 'Easy'].includes(legacy || '')) return '基础'
  if (['中等', 'Medium'].includes(legacy || '')) return '普及'
  if (['困难', 'Hard'].includes(legacy || '')) return '提高'
  if (legacy === '挑战') return '挑战'
  return ''
}

export function problemDifficulty(problem?: Pick<Problem, 'difficulty' | 'tags'> | null) {
  return problem?.difficulty || difficultyFromTags(problem?.tags)
}

export function difficultyTagType(difficulty?: string): 'success' | 'warning' | 'danger' | 'info' {
  if (!difficulty) return 'info'
  if (difficulty === '入门') return 'success'
  if (difficulty === '基础') return 'info'
  if (difficulty === '普及') return 'warning'
  if (difficulty === '提高' || difficulty === '综合' || difficulty === '挑战') return 'danger'
  return 'info'
}

export function difficultyClass(difficulty?: string) {
  const index = problemDifficultyOptions.indexOf(difficulty as typeof problemDifficultyOptions[number])
  return index >= 0 ? `difficulty-level-${index + 1}` : 'difficulty-level-unknown'
}

export function problemMatchesFilters(problem: Problem, filters: ProblemFilters) {
  const keyword = filters.keyword.trim().toLowerCase()
  const tags = tagList(problem.tags)
  if (keyword) {
    const haystack = [String(problem.id), problem.display_code, problem.title, ...tags].filter(Boolean).join(' ').toLowerCase()
    if (!haystack.includes(keyword)) return false
  }
  if (filters.tags.length && !filters.tags.some((tag) => tags.includes(tag))) return false
  if (filters.difficulty && problemDifficulty(problem) !== filters.difficulty) return false
  if (filters.status !== 'all' && problem.progress_status !== filters.status) return false
  return true
}
