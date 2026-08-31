import { describe, expect, it } from "vitest";
import { parseStoredExamDraft } from "./draft";

describe("exam draft storage", () => {
  it("returns a normalized stored draft", () => {
    const draft = parseStoredExamDraft(
      JSON.stringify({
        form: {
          course_id: 2,
          class_id: -1,
          title: "算法设计期中考试",
          description: "",
          starts_at: "2026-09-05T01:00:00.000Z",
          ends_at: "2026-09-05T05:00:00.000Z",
          manual_review: false,
          ranking_visible: true,
          scoring_rule: "score",
        },
        selectedProblems: [
          {
            problem_id: 1,
            title: "A + B",
            source: "题库",
            score: 100,
            label: "A",
          },
        ],
        step: 7,
        savedAt: "2026-08-30T02:23:00.000Z",
      }),
    );

    expect(draft?.form.title).toBe("算法设计期中考试");
    expect(draft?.form.scoring_rule).toBe("ioi");
    expect(draft?.form.freeze_enabled).toBe(false);
    expect(draft?.form.freeze_duration_minutes).toBe(60);
    expect(draft?.selectedProblems).toHaveLength(1);
    expect(draft?.step).toBe(2);
  });

  it("ignores invalid or unrelated local storage values", () => {
    expect(parseStoredExamDraft("{")).toBeNull();
    expect(
      parseStoredExamDraft(JSON.stringify({ form: { title: "缺少题目列表" } })),
    ).toBeNull();
    expect(parseStoredExamDraft(null)).toBeNull();
  });
});
