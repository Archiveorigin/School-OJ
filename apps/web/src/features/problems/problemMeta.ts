import type { Problem } from '../../api/client'

export type ProblemStatusFilter = 'all' | 'unattempted' | 'attempted' | 'accepted'

export interface ProblemSample {
  index: number
  name: string
  input: string
  output: string
}

export interface ProblemFilters {
  keyword: string
  tag: string
  difficulty: string
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
  return problem.display_code || '未编号'
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
  return splitStatementSamples(source).samples
}

export function stripStatementSamples(source?: string | null) {
  return splitStatementSamples(source).statement
}

export function replaceStatementSamples(source: string | null | undefined, samples: Array<Pick<ProblemSample, 'name' | 'input' | 'output'>>) {
  const statement = stripStatementSamples(source)
  const rendered = samples
    .filter((sample) => sample.input.trim() || sample.output.trim())
    .map((sample, index) => {
      const number = index + 1
      const name = sample.name.trim().replace(/[\r\n（）()]/g, ' ') || `样例 ${number}`
      return [
        `### 输入样例 ${number}（${name}）`,
        fencedSample(sample.input),
        `### 输出样例 ${number}（${name}）`,
        fencedSample(sample.output)
      ].join('\n\n')
    })
    .join('\n\n')
  return [statement, rendered].filter(Boolean).join('\n\n').trim()
}

function splitStatementSamples(source?: string | null): { statement: string; samples: ProblemSample[] } {
  if (!source) return { statement: '', samples: [] }
  const lines = source.replace(/\r\n?/g, '\n').split('\n')
  const inputs: Array<{ name: string; value: string }> = []
  const outputs: Array<{ name: string; value: string }> = []
  const removed = new Set<number>()
  for (let i = 0; i < lines.length; i += 1) {
    const label = sampleLabel(lines[i])
    if (!label) continue
    const result = nextCodeBlock(lines, i + 1)
    if (!result) continue
    for (let line = i; line <= result.end; line += 1) removed.add(line)
    if (label.kind === 'input') inputs.push({ name: label.name, value: result.value })
    else outputs.push({ name: label.name, value: result.value })
    i = result.end
  }
  const count = Math.min(inputs.length, outputs.length)
  const samples = Array.from({ length: count }, (_, index) => ({
    index: index + 1,
    name: inputs[index].name || outputs[index].name || `样例 ${index + 1}`,
    input: inputs[index].value,
    output: outputs[index].value
  }))
  const statement = lines
    .filter((_, index) => !removed.has(index))
    .join('\n')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
  return { statement, samples }
}

export function problemMatchesFilters(problem: Problem, filters: ProblemFilters) {
  const keyword = filters.keyword.trim().toLowerCase()
  const tags = tagList(problem.tags)
  if (keyword) {
    const haystack = [String(problem.id), problem.display_code, problem.slug, problem.title, ...tags].filter(Boolean).join(' ').toLowerCase()
    if (!haystack.includes(keyword)) return false
  }
  if (filters.tag && !tags.includes(filters.tag)) return false
  if (filters.difficulty && difficultyFromTags(problem.tags) !== filters.difficulty) return false
  if (filters.status !== 'all' && problem.progress_status !== filters.status) return false
  return true
}

function sampleLabel(line: string): { kind: 'input' | 'output'; name: string } | null {
  const label = line
    .trim()
    .replace(/^#{1,6}\s*/, '')
    .replace(/^(\*\*|__)(.*)(\*\*|__)$/, '$2')
    .replace(/[:：]\s*$/, '')
    .trim()
  if (!label || label.length > 100) return null
  const match = label.match(/^(输入样例|样例输入|输入示例|示例输入|sample\s*input|输出样例|样例输出|输出示例|示例输出|sample\s*output)(?:\s*\d+)?(?:\s*[（(]([^）)]*)[）)])?$/i)
  if (!match) return null
  const kind = /^(输入|样例输入|输入示例|示例输入|sample\s*input)/i.test(match[1]) ? 'input' : 'output'
  return { kind, name: (match[2] || '').trim() }
}

function nextCodeBlock(lines: string[], start: number) {
  for (let i = start; i < lines.length; i += 1) {
    const trimmed = lines[i].trim()
    if (trimmed && !/^(```+|~~~+)/.test(trimmed)) return null
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

function fencedSample(value: string) {
  const normalized = value.replace(/\r\n?/g, '\n').replace(/\n+$/, '')
  const longestRun = Math.max(0, ...(normalized.match(/`+/g) || []).map((run) => run.length))
  const fence = '`'.repeat(Math.max(3, longestRun + 1))
  return `${fence}text\n${normalized}\n${fence}`
}
