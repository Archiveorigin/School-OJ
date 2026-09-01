import type { ProblemChangeAction } from '../../api/client'

export interface ProblemChangeTicketDraft {
  action: ProblemChangeAction
  problemID?: number
  targetScope: string
  teamProblemSetID?: number
  description: string
  attachment: File | null
  overwriteConfirmed?: boolean
}

export function validateProblemChangeTicketDraft(draft: ProblemChangeTicketDraft) {
  if (draft.action !== 'create' && !draft.problemID) return '请选择目标题目'
  if (
    draft.action === 'create' &&
    draft.targetScope === 'team_problem_set' &&
    !draft.teamProblemSetID
  )
    return '请填写团队题单 ID'
  if (draft.action !== 'create' && !draft.overwriteConfirmed) return '请确认采用覆盖性操作'
  if (draft.description.trim().length < 8) return '请至少用 8 个字说明修改需求'
  if (!draft.attachment || draft.attachment.size <= 0) return '请上传修改后的正确内容作为参考附件'
  return ''
}

export function problemChangeOperationMode(action: ProblemChangeAction) {
  return action === 'create' ? 'create' : 'overwrite'
}
