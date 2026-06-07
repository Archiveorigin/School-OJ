import { formatDateTime } from '../time'

export type WorkStatusFilter = 'all' | 'unattempted' | 'unsubmitted' | 'submitted'

export interface AssignmentFilters {
  keyword: string
  status: WorkStatusFilter
}

export const assignmentStatusOptions: Array<{ label: string; value: WorkStatusFilter }> = [
  { label: '全部状态', value: 'all' },
  { label: '未尝试', value: 'unattempted' },
  { label: '未提交', value: 'unsubmitted' },
  { label: '已提交', value: 'submitted' }
]

export function workStatusLabel(status?: string) {
  if (status === 'submitted') return '已提交'
  if (status === 'unsubmitted') return '未提交'
  return '未尝试'
}

export function workStatusType(status?: string): 'success' | 'warning' | 'info' {
  if (status === 'submitted') return 'success'
  if (status === 'unsubmitted') return 'warning'
  return 'info'
}

export function scoreText(row: any) {
  if (row.work_status !== 'submitted') return '-'
  return row.score_ready ? `${row.total_score} / ${row.max_score}` : '计算中'
}

export { formatDateTime }

export function assignmentState(row: any, now = new Date()) {
  const start = row.starts_at ? new Date(row.starts_at) : null
  const due = row.due_at ? new Date(row.due_at) : null
  if (start && start > now) return { label: '未开始', type: 'warning' as const }
  if (due && due < now) return { label: '已截止', type: 'info' as const }
  return { label: '进行中', type: 'success' as const }
}

export function assignmentProblemCount(row: any) {
  return Array.isArray(row.problems) ? row.problems.length : 0
}

export function assignmentMatchesFilters(row: any, filters: AssignmentFilters) {
  const keyword = filters.keyword.trim().toLowerCase()
  if (keyword) {
    const haystack = [row.course_code, row.course_name, row.class_name, row.title, row.description]
      .join(' ')
      .toLowerCase()
    if (!haystack.includes(keyword)) return false
  }
  if (filters.status !== 'all' && row.work_status !== filters.status) return false
  return true
}
