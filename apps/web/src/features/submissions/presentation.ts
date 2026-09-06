export function languageLabel(language: string) {
  return ({ cpp: 'C++17', c: 'C', python: 'Python 3', java: 'Java 21' } as Record<string, string>)[language] || language
}

export function hasEvaluatedResources(status: string) {
  return ['accepted', 'wrong_answer', 'compile_error', 'runtime_error', 'time_limit', 'memory_limit', 'output_limit'].includes(status)
}

export function sourceLineNumbers(source: string) {
  return Array.from({ length: source.split(/\r\n|\r|\n/).length }, (_, index) => index + 1).join('\n')
}

export const shortVerdicts: Record<string, string> = {
  accepted: 'AC', ac: 'AC', wrong_answer: 'WA', compile_error: 'CE', runtime_error: 'RE',
  time_limit: 'TLE', memory_limit: 'MLE', output_limit: 'OLE', system_error: 'SE',
  queued: '排队中', running: '评测中', recorded: '已记录', pending_review: '待评阅', manual_graded: '已评分',
}
