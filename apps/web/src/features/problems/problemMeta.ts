import type { Problem } from '../../api/client'

export type ProblemStatusFilter = 'all' | 'unattempted' | 'attempted' | 'accepted'

export interface ProblemFilters {
  keyword: string
  tag: string
  status: ProblemStatusFilter
}

export const problemStatusOptions: Array<{ label: string; value: ProblemStatusFilter }> = [
  { label: '全部状态', value: 'all' },
  { label: '未尝试', value: 'unattempted' },
  { label: '未通过', value: 'attempted' },
  { label: '已通过', value: 'accepted' }
]

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

export function problemLimitText(problem: Pick<Problem, 'time_limit_ms' | 'memory_limit_mb' | 'output_limit_kb'>) {
  return `${problem.time_limit_ms} ms / ${problem.memory_limit_mb} MB / ${problem.output_limit_kb} KB`
}

export function problemMatchesFilters(problem: Problem, filters: ProblemFilters) {
  const keyword = filters.keyword.trim().toLowerCase()
  const tags = tagList(problem.tags)
  if (keyword) {
    const haystack = [String(problem.id), problem.slug, problem.title, ...tags].join(' ').toLowerCase()
    if (!haystack.includes(keyword)) return false
  }
  if (filters.tag && !tags.includes(filters.tag)) return false
  if (filters.status !== 'all' && problem.progress_status !== filters.status) return false
  return true
}
