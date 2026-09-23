import type {
  ProblemChangeAction,
  ProblemChangeStatus,
} from "../../api/client";

export const ticketActionLabels: Record<ProblemChangeAction, string> = {
  replace: "修改",
  suggest: "建议",
  archive: "删除",
  create: "新增",
};
export const ticketStatusLabels: Record<ProblemChangeStatus, string> = {
  pending: "待处理",
  processing: "处理中",
  completed: "已完成",
  rejected: "已驳回",
  cancelled: "已撤回",
};
export interface ProblemChangeTicketDraft {
  action: ProblemChangeAction;
  problemID?: number;
  eventID?: number;
  targetScope: string;
  teamProblemSetID?: number;
  description: string;
  attachments: File[];
  retainedSize?: number;
  retainedCount?: number;
  draftTitle?: string;
  draftStatement?: string;
}
export function validateProblemChangeTicketDraft(
  draft: ProblemChangeTicketDraft,
) {
  if (draft.action !== "create" && !draft.problemID && !draft.eventID)
    return "请选择目标题目或场次";
  if (draft.action === "archive" && !draft.problemID)
    return "删除申请必须指定题目";
  if (
    draft.action === "create" &&
    draft.targetScope === "team_problem_set" &&
    !draft.teamProblemSetID
  )
    return "请填写团队题单 ID";
  if (
    draft.description.trim().length < 8 ||
    draft.description.trim().length > 10000
  )
    return "修改说明须为 8 至 10000 个字";
  if (
    draft.action === "create" &&
    (!draft.draftTitle?.trim() || !draft.draftStatement?.trim())
  )
    return "请填写题目标题和 Markdown 题面";
  if (draft.attachments.length + (draft.retainedCount || 0) > 20)
    return "最多上传 20 个附件";
  if (
    draft.attachments.some((file) => !file.size || file.size > 32 * 1024 * 1024)
  )
    return "附件须非空，单个不超过 32 MiB";
  if (
    draft.attachments.reduce(
      (sum, file) => sum + file.size,
      draft.retainedSize || 0,
    ) >
    96 * 1024 * 1024
  )
    return "附件总计不能超过 96 MiB";
  return "";
}
export function problemChangeOperationMode(action: ProblemChangeAction) {
  return action === "create" ? "create" : "overwrite";
}
