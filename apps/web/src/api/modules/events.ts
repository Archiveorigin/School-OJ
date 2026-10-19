import { generatedApi, type Problem } from "../client";

export type EventKind = "exam" | "contest";
export type ReviewStatus = "draft" | "pending" | "approved" | "rejected";
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
};
export type PaperProblem = {
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
