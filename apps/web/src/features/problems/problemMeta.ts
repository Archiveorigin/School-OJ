import type { Problem } from '../../api/client'

export type ProblemStatusFilter = 'all' | 'unattempted' | 'attempted' | 'accepted'

export interface ProblemSample {
  index: number
  input: string
  output: string
}

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

export function problemDisplayCode(problem: Pick<Problem, 'id' | 'display_code'>) {
  return problem.display_code || String(problem.id)
}

export function problemLimitText(problem: Pick<Problem, 'time_limit_ms' | 'memory_limit_mb' | 'output_limit_kb'>) {
  return `${problem.time_limit_ms} ms / ${problem.memory_limit_mb} MB / ${problem.output_limit_kb} KB`
}

export function problemLimitLines(problem: Pick<Problem, 'time_limit_ms' | 'memory_limit_mb'>) {
  return [`时间限制：${problem.time_limit_ms} ms`, `内存限制：${problem.memory_limit_mb} MB`]
}

export function difficultyFromTags(tags: unknown) {
  const difficultyTags = ['入门', '简单', '中等', '困难', '挑战', 'Easy', 'Medium', 'Hard']
  const normalized = tagList(tags)
  return normalized.find((tag) => difficultyTags.some((item) => item.toLowerCase() === tag.toLowerCase())) || ''
}

export function difficultyTagType(difficulty?: string): 'success' | 'warning' | 'danger' | 'info' {
  if (!difficulty) return 'info'
  if (['入门', '简单', 'Easy'].some((item) => item.toLowerCase() === difficulty.toLowerCase())) return 'success'
  if (['中等', 'Medium'].some((item) => item.toLowerCase() === difficulty.toLowerCase())) return 'warning'
  if (['困难', '挑战', 'Hard'].some((item) => item.toLowerCase() === difficulty.toLowerCase())) return 'danger'
  return 'info'
}

export function extractStatementSamples(source?: string | null): ProblemSample[] {
  if (!source) return []
  const lines = source.replace(/\r\n?/g, '\n').split('\n')
  const inputs: string[] = []
  const outputs: string[] = []
  for (let i = 0; i < lines.length; i += 1) {
    const kind = sampleLabelKind(lines[i])
    if (!kind) continue
    const result = nextCodeBlock(lines, i + 1)
    if (!result) continue
    if (kind === 'input') inputs.push(result.value)
    else outputs.push(result.value)
    i = result.end
  }
  const count = Math.min(inputs.length, outputs.length)
  return Array.from({ length: count }, (_, index) => ({
    index: index + 1,
    input: inputs[index],
    output: outputs[index]
  }))
}

export function problemMatchesFilters(problem: Problem, filters: ProblemFilters) {
  const keyword = filters.keyword.trim().toLowerCase()
  const tags = tagList(problem.tags)
  if (keyword) {
    const haystack = [String(problem.id), problem.display_code, problem.slug, problem.title, ...tags].filter(Boolean).join(' ').toLowerCase()
    if (!haystack.includes(keyword)) return false
  }
  if (filters.tag && !tags.includes(filters.tag)) return false
  if (filters.status !== 'all' && problem.progress_status !== filters.status) return false
  return true
}

function sampleLabelKind(line: string): 'input' | 'output' | '' {
  const label = line
    .trim()
    .replace(/^#{1,6}\s*/, '')
    .replace(/^(\*\*|__)(.*)(\*\*|__)$/, '$2')
    .replace(/[:：]\s*$/, '')
    .trim()
  if (!label || label.length > 48) return ''
  if (/^(输入样例|样例输入|输入示例|示例输入|sample\s*input)(\s*\d+)?$/i.test(label)) return 'input'
  if (/^(输出样例|样例输出|输出示例|示例输出|sample\s*output)(\s*\d+)?$/i.test(label)) return 'output'
  return ''
}

function nextCodeBlock(lines: string[], start: number) {
  for (let i = start; i < lines.length; i += 1) {
    const trimmed = lines[i].trim()
    const fence = trimmed.match(/^(```+|~~~+)/)
    if (!fence) continue
    const marker = fence[1][0]
    const body: string[] = []
    for (let j = i + 1; j < lines.length; j += 1) {
      if (lines[j].trim().startsWith(marker.repeat(fence[1].length))) {
        return { value: body.join('\n'), end: j }
      }
      body.push(lines[j])
    }
    return null
  }
  return null
}
