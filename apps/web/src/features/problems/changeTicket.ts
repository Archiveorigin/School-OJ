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
  awaiting_changes: "待补充",
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
export function ticketAttachmentLimit(action: ProblemChangeAction) {
  return action === "create" ? 30 : 20;
}
export function validTicketFileSize(file: Pick<File, "name" | "size">) {
  return file.size <= 32 * 1024 * 1024 && (file.size > 0 || file.size === 0 && /\.(in|out)$/i.test(file.name));
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
  const descriptionLength = Array.from(draft.description.trim()).length;
  if (descriptionLength < 4 || descriptionLength > 10000)
    return "修改说明须为 4 至 10000 个字";
  if (
    draft.action === "create" &&
    (!draft.draftTitle?.trim() || !draft.draftStatement?.trim())
  )
    return "请填写题目标题和 Markdown 题面";
  const limit = ticketAttachmentLimit(draft.action);
  if (draft.attachments.length + (draft.retainedCount || 0) > limit)
    return `最多上传 ${limit} 个附件`;
  if (
    draft.attachments.some((file) => !validTicketFileSize(file))
  )
    return "单个附件不超过 32 MiB，仅 .in / .out 测试文件允许为空";
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
