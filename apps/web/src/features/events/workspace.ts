import type { Problem, Submission } from '../../api/client'
export type EventProblem = { problem_id: number; label: string; problem: Problem; score: number; submission_status?: string }
export function scoringDescription(rule: string) {
  return rule === 'oi' ? '最后一次提交计分，结束前隐藏结果' : rule === 'ioi' ? '每题最高得分，总分优先' : '通过题数优先，错误提交计入罚时'
}
export function problemLabel(index: number): string {
  let label = ''
  for (index += 1; index > 0; index = Math.floor((index - 1) / 26)) label = String.fromCharCode(65 + (index - 1) % 26) + label
  return label
}
export function problemPath(base: string, label: string) { return `${base}/problems/${encodeURIComponent(label)}` }
export function examProblemItems(links: Array<{ problem_id: number; label?: string; score: number; problem: Problem }>, latest: Submission[]): EventProblem[] {
  return links.map((link, index) => ({ ...link, label: link.label || problemLabel(index), submission_status: latest.find(row => row.problem_id === link.problem_id)?.status }))
}
export function legacyExamPath(id: string, section: string, query: Record<string, unknown>) {
  const base = `/exams/${id}`
  if (section === 'overview') return `${base}/description`
  if (section === 'ranking') return `${base}/scoreboard`
  if (typeof query.label === 'string' && query.label) return problemPath(base, query.label)
  return `${base}/problems`
}
