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
  scoring_rule: "acm",
  freeze_enabled: false,
  freeze_duration_minutes: 60,
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

  it("validates OI visibility and freeze duration", () => {
    const items = [problem("A", 100)];
    expect(validateExamDraft({ ...draft(), scoring_rule: "oi", ranking_visible: true }, items)).toContain("OI");
    expect(validateExamDraft({ ...draft(), freeze_enabled: true }, items)).toContain("开始和结束时间");
    const startsAt = new Date("2026-08-31T08:00:00Z");
    const endsAt = new Date("2026-08-31T10:00:00Z");
    expect(validateExamDraft({ ...draft(), starts_at: startsAt, ends_at: endsAt, freeze_enabled: true, freeze_duration_minutes: 120 }, items)).toContain("小于考试时长");
  });
});
