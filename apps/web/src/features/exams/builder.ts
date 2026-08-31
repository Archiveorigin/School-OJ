import type { ExamDraft, SelectedExamProblem } from "./types";

export function defaultProblemLabel(index: number) {
  index += 1;
  let label = "";
  while (index > 0) {
    index -= 1;
    label = String.fromCharCode(65 + (index % 26)) + label;
    index = Math.floor(index / 26);
  }
  return label;
}

export function nextProblemLabel(items: SelectedExamProblem[]) {
  const used = new Set(items.map((item) => item.label.trim().toLowerCase()));
  for (let index = 0; index < 702; index += 1) {
    const label = defaultProblemLabel(index);
    if (!used.has(label.toLowerCase())) return label;
  }
  return defaultProblemLabel(items.length);
}

export function hasDuplicateProblemLabels(items: SelectedExamProblem[]) {
  const seen = new Set<string>();
  for (const item of items) {
    const label = item.label.trim().toLowerCase();
    if (seen.has(label)) return true;
    seen.add(label);
  }
  return false;
}

export function examTotalScore(items: SelectedExamProblem[]) {
  return items.reduce((sum, item) => sum + Number(item.score || 0), 0);
}

export function formatExamDate(value: Date | string | null | undefined) {
  if (!value) return "";
  const date = value instanceof Date ? value : new Date(value);
  const pad = (part: number) => String(part).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export function validateExamDraft(
  draft: ExamDraft,
  items: SelectedExamProblem[],
) {
  if (draft.class_id === -1 && !draft.course_id) return "请选择课程";
  if (draft.class_id !== -1 && !draft.class_id) return "请选择班级";
  if (!draft.title.trim()) return "请填写考试标题";
  if (!items.length) return "请至少选择一道题目";
  if (items.some((item) => !item.label.trim())) return "每道题都需要填写题号";
  if (hasDuplicateProblemLabels(items)) return "题号不能重复";
  if (items.some((item) => item.release_after_exam) && !draft.ends_at) {
    return "使用预备题或考试内新建题时必须填写结束时间";
  }
  if (draft.scoring_rule === "oi" && (draft.ranking_visible || draft.freeze_enabled)) {
    return "OI 赛制不能开启实时榜单或封榜";
  }
  if (draft.freeze_enabled) {
    if (!draft.starts_at || !draft.ends_at) return "封榜考试必须设置开始和结束时间";
    const duration = (new Date(draft.ends_at).getTime() - new Date(draft.starts_at).getTime()) / 60000;
    if (duration <= 0) return "结束时间必须晚于开始时间";
    if (draft.freeze_duration_minutes < 1 || draft.freeze_duration_minutes >= duration) {
      return "封榜分钟数必须小于考试时长";
    }
  }
  return "";
}
