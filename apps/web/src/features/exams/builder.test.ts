import { describe, expect, it } from "vitest";
import {
  defaultProblemLabel,
  examTotalScore,
  hasDuplicateProblemLabels,
  nextProblemLabel,
  validateExamDraft,
} from "./builder";
import type { ExamDraft, SelectedExamProblem } from "./types";

const problem = (label: string, score = 10): SelectedExamProblem => ({
  problem_id: label.charCodeAt(0),
  title: label,
  source: "题库",
  score,
  label,
});

const draft = (): ExamDraft => ({
  course_id: 1,
  class_id: -1,
  title: "算法设计期中考试",
  description: "",
  starts_at: null,
  ends_at: null,
  manual_review: false,
  ranking_visible: false,
  scoring_rule: "penalty",
});

describe("exam builder", () => {
  it("creates spreadsheet-style labels", () => {
    expect(defaultProblemLabel(0)).toBe("A");
    expect(defaultProblemLabel(25)).toBe("Z");
    expect(defaultProblemLabel(26)).toBe("AA");
  });

  it("finds the next unused label and detects duplicates case-insensitively", () => {
    expect(nextProblemLabel([problem("A"), problem("C")])).toBe("B");
    expect(hasDuplicateProblemLabels([problem("A"), problem("a")])).toBe(true);
  });

  it("calculates total score and validates release timing", () => {
    const items = [
      problem("A", 40),
      { ...problem("B", 60), release_after_exam: true },
    ];
    expect(examTotalScore(items)).toBe(100);
    expect(validateExamDraft(draft(), items)).toContain("结束时间");
    expect(validateExamDraft({ ...draft(), ends_at: new Date() }, items)).toBe(
      "",
    );
  });
});
