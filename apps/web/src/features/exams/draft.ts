import type { ExamDraft, SelectedExamProblem } from "./types";

export const examDraftStorageKey = "school-oj-exam-create-draft";

export type StoredExamDraft = {
  form: ExamDraft;
  selectedProblems: SelectedExamProblem[];
  step: number;
  savedAt?: string;
};

export function parseStoredExamDraft(
  raw: string | null | undefined,
): StoredExamDraft | null {
  if (!raw) return null;
  try {
    const value = JSON.parse(raw) as Partial<StoredExamDraft>;
    if (
      !value.form ||
      typeof value.form.title !== "string" ||
      !Array.isArray(value.selectedProblems)
    )
      return null;
    return {
      form: value.form,
      selectedProblems: value.selectedProblems,
      step: Math.min(2, Math.max(0, Number(value.step) || 0)),
      savedAt: typeof value.savedAt === "string" ? value.savedAt : undefined,
    };
  } catch {
    return null;
  }
}
