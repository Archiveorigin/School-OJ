import { describe, expect, it } from 'vitest'
import { validateProblemChangeTicketDraft, problemChangeOperationMode } from '../src/features/problems/changeTicket'
const base = { action: 'replace' as const, problemID: 2, targetScope: 'public', description: '修正题目内容并补充测试数据', attachments: [] as File[] }
describe('ticket feedback validation', () => {
  it('accepts feedback without attachments or an overwrite confirmation', () => {
    for (const action of ['replace', 'suggest', 'archive'] as const) expect(validateProblemChangeTicketDraft({ ...base, action })).toBe('')
  })
  it('requires both the new title and preview source', () => {
    expect(validateProblemChangeTicketDraft({ ...base, action: 'create', problemID: undefined })).toContain('Markdown')
    expect(validateProblemChangeTicketDraft({ ...base, action: 'create', problemID: undefined, draftTitle: '新题目', draftStatement: '## 描述' })).toBe('')
  })
  it('accepts event feedback and refuses a targetless delete request', () => {
    expect(validateProblemChangeTicketDraft({ ...base, problemID: undefined, eventID: 3 })).toBe('')
    expect(validateProblemChangeTicketDraft({ ...base, problemID: undefined, eventID: 3, action: 'archive' })).toContain('指定题目')
  })
  it('counts retained and newly uploaded files together', () => {
    const file = new File(['1 2'], '1.in')
    expect(validateProblemChangeTicketDraft({ ...base, attachments: [file], retainedCount: 20 })).toContain('20')
    expect(validateProblemChangeTicketDraft({ ...base, attachments: [file], retainedSize: 96 * 1024 * 1024 })).toContain('96')
    expect(validateProblemChangeTicketDraft({ ...base, attachments: [new File([], 'empty.in')] })).toContain('非空')
    expect(validateProblemChangeTicketDraft({ ...base, attachments: [file, new File(['3'], '1.out')] })).toBe('')
  })
  it('keeps explicit overwrite mode for administrators applying packages', () => {
    expect(problemChangeOperationMode('replace')).toBe('overwrite')
    expect(problemChangeOperationMode('archive')).toBe('overwrite')
    expect(problemChangeOperationMode('create')).toBe('create')
  })
})
