import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const problemDetailSource = readFileSync(
  resolve(process.cwd(), 'src/views/problems/ProblemDetail.vue'),
  'utf8',
)
const problemSubmissionsSource = readFileSync(
  resolve(process.cwd(), 'src/views/problems/ProblemSubmissions.vue'),
  'utf8',
)
const submissionsSource = readFileSync(
  resolve(process.cwd(), 'src/views/Submissions.vue'),
  'utf8',
)
const submissionDetailDialogSource = readFileSync(
  resolve(process.cwd(), 'src/components/SubmissionDetailDialog.vue'),
  'utf8',
)

describe('problem submission UI regressions', () => {
  it('does not render the standalone result panel below a problem', () => {
    expect(problemDetailSource).not.toContain('class="panel submission-result"')
    expect(problemDetailSource).not.toMatch(/\.submission-result\s*\{/)
    expect(problemDetailSource).toContain(':status="live?.status"')
    expect(problemDetailSource).toContain(':message="live?.message"')
  })

  it('uses the shared detail dialog in both submission record pages', () => {
    expect(problemSubmissionsSource).toContain('<SubmissionDetailDialog')
    expect(submissionsSource).toContain('<SubmissionDetailDialog')
    expect(problemSubmissionsSource).not.toContain('<el-dialog')
    expect(submissionsSource).not.toContain('<el-dialog')
  })

  it('uses the shared svg mark and never renders message or result details', () => {
    expect(submissionDetailDialogSource).toContain('<SubmissionStatusMark')
    expect(submissionDetailDialogSource).toContain(
      ':status="submission.status"',
    )
    expect(submissionDetailDialogSource).not.toContain('student_no')
    expect(submissionDetailDialogSource).not.toContain('<el-alert')
    expect(submissionDetailDialogSource).not.toContain('<el-collapse')
    expect(submissionDetailDialogSource).not.toContain('submission.message')
    expect(submissionDetailDialogSource).not.toContain('detail.results')
    expect(submissionsSource).not.toContain('detail.submission.message')
    expect(submissionsSource).not.toContain('detail.results')
  })

  it('keeps the four-card desktop layout and one-column mobile layout', () => {
    expect(submissionDetailDialogSource).toContain(
      'width="min(1200px, calc(100vw - 28px))"',
    )
    expect(submissionDetailDialogSource).toContain(
      'grid-template-columns: repeat(4, minmax(0, 1fr))',
    )
    expect(submissionDetailDialogSource).toContain('min-height: 350px')
    expect(submissionDetailDialogSource).toMatch(
      /@media \(max-width: 760px\)[\s\S]*\.submission-detail-summary\s*\{\s*grid-template-columns:\s*1fr;/,
    )
    expect(submissionDetailDialogSource).toMatch(
      /@media \(max-width: 760px\)[\s\S]*\.submission-source-code\s*\{\s*min-height:\s*260px;/,
    )
  })
})
