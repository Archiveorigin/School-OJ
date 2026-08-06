import { beforeEach, describe, expect, it } from 'vitest'
import { clearSubmissionDraft, loadSubmissionDraft, saveSubmissionDraft, submissionDraftKey } from './drafts'

const scope = { userId: 27, resourceType: 'contest' as const, resourceId: 9, problemId: 3 }

describe('submission drafts', () => {
  beforeEach(() => localStorage.clear())

  it('isolates drafts by context, problem and language', () => {
    saveSubmissionDraft(scope, 'cpp', 'int main() {}')
    expect(loadSubmissionDraft(scope, 'cpp')).toBe('int main() {}')
    expect(loadSubmissionDraft(scope, 'python')).toBeNull()
    expect(submissionDraftKey(scope, 'cpp')).toContain('user-27:contest:9:3:cpp')

    const anotherUser = { ...scope, userId: 28 }
    expect(loadSubmissionDraft(anotherUser, 'cpp')).toBeNull()
  })

  it('clears a successful submission draft only', () => {
    saveSubmissionDraft(scope, 'cpp', 'cpp source')
    saveSubmissionDraft(scope, 'python', 'python source')
    clearSubmissionDraft(scope, 'cpp')
    expect(loadSubmissionDraft(scope, 'cpp')).toBeNull()
    expect(loadSubmissionDraft(scope, 'python')).toBe('python source')
  })
})
