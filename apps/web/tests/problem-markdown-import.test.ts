import { describe, expect, it } from 'vitest'
import { importProblemMarkdown } from '../src/features/problems/importMarkdown'
import { parseStatement, serializeStatement, validateStatement } from '../src/features/problems/statementTemplate'

const source = '# 两数之和\n\n## 题目描述\n\n计算 $a+b$，满足\n\n$$a \\le b$$\n\n## 输入格式\n\n两个整数\n\n## 输出格式\n\n输出和\n\n## 样例输入 1\n\n```text\n  1 2\n```\n\n## 样例输出 1\n\n```text\n3\n```'
describe('problem Markdown import', () => {
  it('preserves formulas and sample spacing while omitting an empty hints section', () => {
    const result = importProblemMarkdown(source), parsed = parseStatement(result.statement)
    expect(result.title).toBe('两数之和')
    expect(result.warnings).toEqual([])
    expect(result.statement).toContain('$$a \\le b$$')
    expect(result.statement).not.toContain('## 数据范围与提示')
    expect(parsed.legacy).toBe(false)
    expect(parsed.draft.samples[0].input).toBe('  1 2')
    expect(validateStatement(parsed.draft)).toBe('')
  })
  it('round trips the prescribed format with multiple samples and optional hints', () => {
    const parsed = parseStatement(importProblemMarkdown(source).statement).draft
    parsed.hints = '0 ≤ a,b ≤ 100'
    parsed.samples.push({ input: '## not a heading\n', output: '7\n', explanation: '第二组' })
    const result = importProblemMarkdown('# 标题\n\n' + serializeStatement(parsed))
    expect(parseStatement(result.statement).draft).toEqual(parsed)
  })
  it('loads earlier drafts with an empty hints heading without treating them as legacy text', () => {
    const result = importProblemMarkdown(source)
    const parsed = parseStatement(result.statement + '\n\n## 数据范围与提示')
    expect(parsed.legacy).toBe(false)
    expect(parsed.draft.hints).toBe('')
  })
  it('preserves unfamiliar subsections and rejects missing or multiple titles', () => {
    const result = importProblemMarkdown(source.replace('计算 $a+b$', '### 子任务\n\n计算 $a+b$'))
    expect(result.statement).toContain('### 子任务')
    expect(() => importProblemMarkdown(source.replace('# 两数之和\n', ''))).toThrow('一级标题')
    expect(() => importProblemMarkdown(source + '\n\n# 第二题')).toThrow('一级标题')
    expect(() => importProblemMarkdown('# 题\n\n' + '中'.repeat(70000))).toThrow('200000')
  })
})
