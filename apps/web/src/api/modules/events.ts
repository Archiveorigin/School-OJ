import { generatedApi, type Problem } from "../client";
import type { StatementSections } from "../../features/events/statement";

export type EventKind = "exam" | "contest";
export type ReviewStatus = "draft" | "pending" | "approved" | "rejected";
export type EventLifecycle =
  | "draft"
  | "pending"
  | "rejected"
  | "upcoming"
  | "running"
  | "closed"
  | "completed";
export const lifecycleLabels: Record<EventLifecycle, string> = {
  draft: "草稿",
  pending: "待审核",
  rejected: "已退回",
  upcoming: "未开始",
  running: "进行中",
  closed: "已截止",
  completed: "已完结",
};
export function eventLifecycle(
  event: {
    review_status?: string;
    starts_at?: string;
    ends_at?: string;
    duration_minutes?: number;
    completed_at?: string | null;
  },
  now = Date.now(),
): EventLifecycle {
  if (event.review_status && event.review_status !== "approved")
    return event.review_status as EventLifecycle;
  const start = event.starts_at ? Date.parse(event.starts_at) : 0;
  const end = event.ends_at
    ? Date.parse(event.ends_at)
    : start && event.duration_minutes
      ? start + event.duration_minutes * 60000
      : 0;
  if (start > now) return "upcoming";
  if (end && end <= now) return event.completed_at ? "completed" : "closed";
  return "running";
}
export type EventRecord = {
  id: number;
  title: string;
  description: string;
  starts_at?: string;
  ends_at?: string;
  duration_minutes?: number;
  course_id?: number;
  team_id?: number;
  scoring_rule: string;
  review_status: ReviewStatus;
  review_revision: number;
  review_note: string;
  completed_at?: string | null;
  completion_note?: string;
  lifecycle?: EventLifecycle;
};
export type PaperProblem = {
  statement_sections?: StatementSections;
  problem_id: number;
  problem_version_id: number;
  label: string;
  score: number;
  problem: Problem;
  case_count: number;
  checker: {
    type: "exact" | "tokens" | "float";
    absolute_tolerance?: number;
    relative_tolerance?: number;
  };
};
export type Paper = {
  event: EventRecord;
  kind: EventKind;
  problems: PaperProblem[];
  can_edit: boolean;
  lifecycle?: EventLifecycle;
  completion_checks?: { judging: number; ungraded: number };
  can_view_ranking?: boolean;
  history?: Array<{
    id: number;
    action: string;
    revision: number;
    note: string;
    created_at: string;
  }>;
};
export type EventFile = {
  id: number;
  name: string;
  size: number;
  created_at: string;
};
export type EventListItem = EventRecord & {
  kind: EventKind;
  context_name: string;
  problem_count: number;
};
export const reviewLabels: Record<ReviewStatus, string> = {
  draft: "草稿",
  pending: "待审核",
  approved: "已批准",
  rejected: "已退回",
};
export const eventPath = (kind: EventKind, id: number) => ({
  kind,
  event_id: id,
});
export const eventAssets = (kind: EventKind, id: number, problem: number) =>
  `/event-papers/${kind}/${id}/problems/${problem}/assets`;
export async function loadPaper(kind: EventKind, id: number, archive = false) {
  const request = { path: eventPath(kind, id) };
  return (
    archive
      ? await generatedApi.getEventArchive<Paper>(request)
      : await generatedApi.getEventPaper<Paper>(request)
  ).data;
}
export async function listEvents(
  archive: boolean,
  query: { page: number; q: string; kind: string; status?: string },
) {
  const request = { query };
  return (
    archive
      ? await generatedApi.listEventArchives<{
          items: EventListItem[];
          total: number;
        }>(request)
      : await generatedApi.listEventReviews<{
          items: EventListItem[];
          total: number;
        }>(request)
  ).data;
}
export async function downloadEventFile(
  kind: EventKind,
  id: number,
  file: EventFile,
) {
  const { data } = await generatedApi.downloadEventAttachment<Blob>({
    path: { ...eventPath(kind, id), attachment_id: file.id },
    responseType: "blob",
  });
  saveDownload(data, file.name);
}
export async function downloadPaperProblem(
  kind: EventKind,
  id: number,
  link: PaperProblem,
) {
  const { data } = await generatedApi.downloadEventProblemPackage<Blob>({
    path: { ...eventPath(kind, id), problem_id: link.problem_id },
    responseType: "blob",
  });
  saveDownload(data, `${link.label}-v${link.problem_version_id}.zip`);
}
function saveDownload(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob),
    a = document.createElement("a");
  a.href = url;
  a.download = name;
  document.body.append(a);
  a.click();
  a.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 30000);
}
