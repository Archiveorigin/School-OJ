import { adaptTeamContestRanking } from "./adapters";
import type { LeaderboardData } from "./types";
export type ReplayPacket = {
  schema_version: 1;
  kind: "contest-replay";
  contest: Record<string, any>;
  problems: any[];
  initial_rows: any[];
  final_rows: any[];
  events: any[];
  freeze_at?: string;
};
export function parseReplay(text: string): ReplayPacket {
  if (text.length > 32 * 1024 * 1024) throw new Error("滚榜数据包超过 32 MiB");
  const data = JSON.parse(text);
  if (
    data.schema_version !== 1 ||
    data.kind !== "contest-replay" ||
    !data.contest ||
    !Array.isArray(data.problems) ||
    !Array.isArray(data.initial_rows) ||
    !Array.isArray(data.final_rows) ||
    data.final_rows.length > 100000 ||
    data.initial_rows.length > 100000
  )
    throw new Error("不支持的滚榜数据包");
  const initial = new Set(data.initial_rows.map((r: any) => r.user_id));
  if (
    initial.size !== data.initial_rows.length ||
    initial.size !== data.final_rows.length ||
    new Set(data.final_rows.map((r: any) => r.user_id)).size !== initial.size ||
    data.initial_rows.some((r: any) => !Array.isArray(r.problems)) ||
    data.final_rows.some(
      (r: any) => !initial.has(r.user_id) || !Array.isArray(r.problems),
    )
  )
    throw new Error("滚榜参赛者或题目数据不完整");
  return data;
}
// Materialize only the displayed standings; memory stays proportional to one board.
export function createContestReplay(packet: ReplayPacket) {
  const rows = structuredClone(packet.initial_rows);
  const final = new Map(packet.final_rows.map((row) => [row.user_id, row]));
  const byId = new Map(rows.map((row) => [row.user_id, row]));
  const pending = new Map<number, Map<number, any>>();
  for (const row of rows) {
    const target = final.get(row.user_id);
    const changes = new Map<number, any>();
    for (const cell of row.problems) {
      const end = target.problems.find(
        (c: any) => c.problem_id === cell.problem_id,
      );
      if (end && JSON.stringify(cell) !== JSON.stringify(end))
        changes.set(cell.problem_id, end);
    }
    if (changes.size) pending.set(row.user_id, changes);
  }
  const total = [...pending.values()].reduce(
    (sum, cells) => sum + cells.size,
    0,
  );
  let step = 0;
  let consistencyError = "",
    checked = false;
  let focus: { userId: number; problemId: number } | undefined;
  const snapshot = (): LeaderboardData => {
    if (step === total && !checked) {
      checked = true;
      for (const row of rows) {
        const expected = final.get(row.user_id);
        if (
          !expected ||
          row.solved !== expected.solved ||
          row.total_score !== expected.total_score ||
          (packet.contest.scoring_rule === "acm" &&
            row.penalty_minutes !== expected.penalty_minutes) ||
          JSON.stringify(row.problems) !== JSON.stringify(expected.problems)
        ) {
          consistencyError = "回放结果与最终榜单不一致，请重新加载核对。";
          break;
        }
      }
      if (consistencyError) console.warn(consistencyError);
    }
    const visible = rows.map((row) => ({
      ...row,
      problems: row.problems.map((cell: any) =>
        pending.get(row.user_id)?.has(cell.problem_id) &&
        cell.status !== "accepted" &&
        (cell.best_score || 0) < 100
          ? { ...cell, status: "frozen" }
          : cell,
      ),
    }));
    const board = adaptTeamContestRanking({
      contest: packet.contest,
      problems: packet.problems,
      rows: step === total && !consistencyError ? packet.final_rows : visible,
      final_ready: true,
    });
    board.replaying = true;
    if (consistencyError) board.consistencyError = consistencyError;
    board.revealUserId = focus?.userId;
    board.revealProblemId = focus?.problemId;
    if (step !== total && packet.freeze_at && packet.contest.starts_at)
      board.currentTimeSeconds = Math.max(
        0,
        Math.min(
          board.durationSeconds,
          (Date.parse(packet.freeze_at) -
            Date.parse(packet.contest.starts_at)) /
            1000,
        ),
      );
    const tieTime = (id: number | string) => {
      const raw = (step === total && !consistencyError ? final : byId).get(
        Number(id),
      );
      const key =
        board.scoringRule === "penalty" ? "solved_at" : "ranking_time";
      const times = raw.problems
        .map((c: any) => Date.parse(c[key] || ""))
        .filter(Number.isFinite);
      return times.length ? Math.max(...times) : Infinity;
    };
    const compare = (
      a: (typeof board.rows)[number],
      b: (typeof board.rows)[number],
    ) => {
      const grouped = Number(Boolean(a.unranked)) - Number(Boolean(b.unranked));
      const performance =
        board.scoringRule === "penalty"
          ? b.solved - a.solved || a.metric - b.metric
          : b.metric - a.metric || b.solved - a.solved;
      if (grouped || performance) return grouped || performance;
      if (packet.contest.scoring_rule === "oi") return 0;
      const left = tieTime(a.id),
        right = tieTime(b.id);
      return left === right ? 0 : left < right ? -1 : 1;
    };
    board.rows.sort(
      (a, b) =>
        compare(a, b) ||
        (a.name === b.name
          ? Number(a.id) - Number(b.id)
          : a.name < b.name
            ? -1
            : 1),
    );
    let count = 0,
      rank = 0;
    let prior: (typeof board.rows)[number] | undefined;
    for (const row of board.rows) {
      if (row.unranked) {
        row.rank = 0;
        continue;
      }
      count++;
      if (!prior || compare(prior, row) !== 0) rank = count;
      if (step !== total) row.rank = rank;
      prior = row;
    }
    board.participantCount = count;
    return board;
  };
  return {
    total,
    get step() {
      return step;
    },
    snapshot,
    next(): LeaderboardData {
      const current = snapshot();
      // Start from the currently lowest remaining contestant, then recalculate
      // their position after every cell, rather than using the final order.
      const target = [...current.rows]
        .reverse()
        .find((row) => pending.has(Number(row.id)));
      if (!target) return current;
      const changes = pending.get(Number(target.id))!;
      const [problemId, cell] = changes.entries().next().value!;
      const row = byId.get(Number(target.id));
      row.problems[
        row.problems.findIndex((c: any) => c.problem_id === problemId)
      ] = structuredClone(cell);
      changes.delete(problemId);
      if (!changes.size) pending.delete(Number(target.id));
      row.solved = row.problems.filter(
        (c: any) => c.solved || c.status === "accepted" || c.best_score >= 100,
      ).length;
      row.total_score = row.problems.reduce(
        (sum: number, c: any) => sum + (c.best_score || 0),
        0,
      );
      row.penalty_minutes = row.problems.reduce(
        (sum: number, c: any) =>
          sum +
          (c.solved || c.status === "accepted"
            ? (c.elapsed_minutes || 0) + (c.wrong_attempts || 0) * 20
            : 0),
        0,
      );
      row.submission_count = row.problems.reduce(
        (sum: number, c: any) => sum + (c.submission_count ?? c.attempts ?? 0),
        0,
      );
      focus = { userId: Number(target.id), problemId };
      step++;
      return snapshot();
    },
  };
}
// Small fixture helper. The interactive screen uses the bounded engine above.
export function replayFrames(packet: ReplayPacket): LeaderboardData[] {
  const replay = createContestReplay(packet);
  const frames = [replay.snapshot()];
  while (replay.step < replay.total) frames.push(replay.next());
  return frames;
}
