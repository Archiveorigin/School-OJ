import { describe, it, expect } from "vitest";
import {
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
