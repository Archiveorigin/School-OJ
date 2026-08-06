export type SubmissionDraftScope = {
  userId: number
  resourceType: 'problem' | 'assignment' | 'exam' | 'contest' | 'problem-set'
  resourceId: number
  problemId: number
}

type StoredDraft = {
  source: string
  updatedAt: string
}

const prefix = 'school-oj-submission-draft:v1'

export function submissionDraftKey(scope: SubmissionDraftScope, language: string) {
  return [prefix, `user-${scope.userId}`, scope.resourceType, scope.resourceId, scope.problemId, language || 'cpp'].join(':')
}

export function loadSubmissionDraft(scope: SubmissionDraftScope, language: string): string | null {
  try {
    const raw = localStorage.getItem(submissionDraftKey(scope, language))
    if (!raw) return null
    const parsed = JSON.parse(raw) as StoredDraft
    return typeof parsed.source === 'string' ? parsed.source : null
  } catch {
    return null
  }
}

export function saveSubmissionDraft(scope: SubmissionDraftScope, language: string, source: string) {
  const key = submissionDraftKey(scope, language)
  if (!source) {
    localStorage.removeItem(key)
    return
  }
  const value: StoredDraft = { source, updatedAt: new Date().toISOString() }
  localStorage.setItem(key, JSON.stringify(value))
}

export function clearSubmissionDraft(scope: SubmissionDraftScope, language: string) {
  localStorage.removeItem(submissionDraftKey(scope, language))
}
