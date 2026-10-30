import { describe, it, expect, vi } from "vitest";
import {
  createContestReplay,
  parseReplay,
  replayFrames,
  type ReplayPacket,
} from "../src/features/leaderboard/replay";
const cell = (status: string, attempts: number, time: number) => ({
  problem_id: 1,
  status,
  attempts,
  elapsed_minutes: time,
  wrong_attempts: status === "accepted" ? 1 : 0,
  best_score: status === "accepted" ? 100 : 0,
});
const row = (id: number, c: any, unranked = false) => ({
  user_id: id,
  name: `选手${id}`,
  rank: id,
  solved: c.status === "accepted" ? 1 : 0,
  penalty_minutes: c.status === "accepted" ? c.elapsed_minutes + 20 : 0,
  total_score: c.best_score,
  submission_count: c.attempts,
  problems: [c],
  unranked,
});
describe("offline replay", () => {
  it("reveals actual cells, changes ranks and keeps out-of-competition status", () => {
    const packet: ReplayPacket = {
      schema_version: 1,
      kind: "contest-replay",
      contest: { scoring_rule: "acm", duration_minutes: 120 },
      problems: [{ problem_id: 1, label: "A" }],
      initial_rows: [
        row(1, cell("wrong_answer", 1, 10)),
        row(2, cell("wrong_answer", 1, 12), true),
      ],
      final_rows: [
        row(1, cell("accepted", 2, 30)),
        row(2, cell("accepted", 2, 20), true),
      ],
      events: [],
    };
    const frames = replayFrames(parseReplay(JSON.stringify(packet)));
    expect(frames[0].rows.find((r) => r.id === 1)?.solved).toBe(0);
    expect(frames.at(-1)?.rows.find((r) => r.id === 1)?.metric).toBe(50);
    expect(frames.at(-1)?.rows.find((r) => r.id === 2)?.unranked).toBe(true);
    expect(packet.initial_rows[0].problems[0].status).toBe("wrong_answer");
  });
  it("rejects unsupported data and inconsistent participants", () => {
    expect(() => parseReplay("{}")).toThrow();
    expect(() =>
      parseReplay(
        JSON.stringify({
          schema_version: 1,
          kind: "contest-replay",
          contest: {},
          problems: [],
          initial_rows: [],
          final_rows: [{ user_id: 9, problems: [] }],
        }),
      ),
    ).toThrow();
  });
});

it("loads large contests without precomputing all board frames and reveals the currently lowest row", () => {
  const initial = Array.from({ length: 1000 }, (_, i) =>
    row(i + 1, cell("wrong_answer", 1, 10)),
  );
  const final = initial.map((r, i) =>
    row(r.user_id, cell("accepted", 2, 1000 - i)),
  );
  const packet: ReplayPacket = {
    schema_version: 1,
    kind: "contest-replay",
    contest: { scoring_rule: "acm", duration_minutes: 1200 },
    problems: [{ problem_id: 1, label: "A" }],
    initial_rows: initial,
    final_rows: final,
    events: [],
  };
  const replay = createContestReplay(packet);
  expect(replay.total).toBe(1000);
  const lowest = replay.snapshot().rows.at(-1)!.id;
  const next = replay.next();
  expect(next.revealUserId).toBe(lowest);
  expect(next.rows.find((r) => r.id === lowest)?.solved).toBe(1);
  expect(initial.find((r) => r.user_id === lowest)!.solved).toBe(0);
});

it("uses exact acceptance times to resolve tied partial penalties", () => {
  const first = {
    ...cell("accepted", 1, 10),
    solved_at: "2026-10-30T10:10:40Z",
  };
  const earlier = {
    ...cell("accepted", 1, 10),
    solved_at: "2026-10-30T10:10:10Z",
  };
  const start = row(1, first),
    other = row(2, earlier);
  const packet: ReplayPacket = {
    schema_version: 1,
    kind: "contest-replay",
    contest: { scoring_rule: "acm", duration_minutes: 60 },
    problems: [{ problem_id: 1, label: "A" }],
    initial_rows: [start, other],
    final_rows: [
      { ...other, rank: 1 },
      { ...start, rank: 2, problems: [{ ...first, submission_count: 3 }] },
    ],
    events: [],
  };
  const replay = createContestReplay(packet),
    initial = replay.snapshot();
  expect(initial.rows.map((r) => r.id)).toEqual([2, 1]);
  expect(initial.rows.map((r) => r.rank)).toEqual([1, 2]);
});

it("reports inconsistent final totals rather than silently replacing the replay", () => {
  const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
  const packet: ReplayPacket = {
    schema_version: 1,
    kind: "contest-replay",
    contest: { scoring_rule: "acm" },
    problems: [{ problem_id: 1, label: "A" }],
    initial_rows: [row(1, cell("wrong_answer", 1, 10))],
    final_rows: [{ ...row(1, cell("accepted", 2, 30)), solved: 9 }],
    events: [],
  };
  const replay = createContestReplay(packet);
  const end = replay.next();
  expect(end.consistencyError).toContain("不一致");
  expect(end.rows[0].solved).toBe(1);
  expect(warn).toHaveBeenCalledTimes(1);
  warn.mockRestore();
});
