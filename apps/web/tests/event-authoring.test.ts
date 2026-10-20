// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
vi.mock('../src/api/client', () => ({ generatedApi: {} }))
import { eventLifecycle } from '../src/api/modules/events'
import { emptyStatement, renderStatement, validateStatement } from '../src/features/events/statement'
import { previewTestFile, testFileError, TEST_PREVIEW_BYTES } from '../src/features/events/testUploads'

describe('event completion and structured authoring', () => {
  it('keeps elapsed events closed until an administrator completes them', () => {
    const event = {review_status: 'approved', starts_at: '2026-01-01T10:00:00Z', duration_minutes: 60}
    expect(eventLifecycle(event, Date.parse('2026-01-01T09:59:00Z'))).toBe('upcoming')
    expect(eventLifecycle(event, Date.parse('2026-01-01T10:00:00Z'))).toBe('running')
    expect(eventLifecycle(event, Date.parse('2026-01-01T11:00:00Z'))).toBe('closed')
    expect(eventLifecycle({...event, completed_at: '2026-01-01T12:00:00Z'}, Date.parse('2026-01-01T12:00:00Z'))).toBe('completed')
    expect(eventLifecycle({...event, review_status: 'pending'}, Date.parse('2026-01-02T00:00:00Z'))).toBe('pending')
  })
  it('requires structured fields and omits empty optional notes', () => {
    const statement = emptyStatement()
    expect(validateStatement(statement)).not.toBe('')
    Object.assign(statement, {description: 'Double', input_format: 'Integer', output_format: 'Integer', samples: [{input: '1', output: '2\n```'}]})
    expect(validateStatement(statement)).toBe('')
    expect(renderStatement(statement)).not.toContain('## 说明')
    expect(renderStatement(statement)).toContain('````text\n2\n```\n````')
    statement.notes = 'Example explanation'
    expect(renderStatement(statement)).toContain('## 说明\n\nExample explanation')
  })
  it('reads only a bounded preview of a 100 KiB test without altering its file', async () => {
    const file = new File(['x'.repeat(100 * 1024)], 'large.in')
    const slice = vi.spyOn(file, 'slice')
    const preview = await previewTestFile(file)
    expect(slice).toHaveBeenCalledWith(0, TEST_PREVIEW_BYTES)
    expect(preview.text).toHaveLength(TEST_PREVIEW_BYTES)
    expect(preview.truncated).toBe(true)
    expect(file.size).toBe(100 * 1024)
  })
  it('rejects missing pairs and conflicting filenames before upload', () => {
    const file = (name: string) => new File(['1'], name)
    expect(testFileError([file('1.in')])).toContain('输出')
    expect(testFileError([file('1.out')])).toContain('输入')
    expect(testFileError([file('1.in'), file('1.out')])).toBe('')
    expect(testFileError([file('1.in'), file('1.IN')])).toContain('重名')
    expect(testFileError([file('unexpected.exe')])).toContain('请选择')
  })
})
