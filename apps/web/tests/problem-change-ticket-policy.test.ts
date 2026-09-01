import { describe, expect, it } from 'vitest'
import {
  problemChangeOperationMode,
  validateProblemChangeTicketDraft
} from '../src/features/problems/changeTicket'

const attachment = new File(['corrected content'], 'corrected.zip', {
  type: 'application/zip'
})

describe('problem change ticket policy', () => {
  it('requires a non-empty corrected-content attachment for every action', () => {
    for (const action of ['create', 'replace', 'archive'] as const) {
      expect(
        validateProblemChangeTicketDraft({
          action,
          problemID: action === 'create' ? undefined : 2,
          targetScope: 'public',
          description: '修正题目内容并补充测试数据',
          attachment: null,
          overwriteConfirmed: action === 'create' ? undefined : true
        })
      ).toContain('参考附件')
      expect(
        validateProblemChangeTicketDraft({
          action,
          problemID: action === 'create' ? undefined : 2,
          targetScope: 'public',
          description: '修正题目内容并补充测试数据',
          attachment,
          overwriteConfirmed: action === 'create' ? undefined : true
        })
      ).toBe('')
    }
  })

  it('uses overwrite mode for replacement and archive operations', () => {
    expect(problemChangeOperationMode('create')).toBe('create')
    expect(problemChangeOperationMode('replace')).toBe('overwrite')
    expect(problemChangeOperationMode('archive')).toBe('overwrite')
  })

  it('requires an explicit overwrite confirmation for replacement and archive tickets', () => {
    const base = {
      action: 'replace' as const,
      problemID: 2,
      targetScope: 'public',
      description: '替换题面并修正所有测试点数据',
      attachment: new File(['corrected'], 'corrected.zip')
    }
    expect(validateProblemChangeTicketDraft(base)).toContain('覆盖性操作')
    expect(validateProblemChangeTicketDraft({ ...base, overwriteConfirmed: true })).toBe('')
  })
})
