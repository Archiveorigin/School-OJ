import { adaptTeamContestRanking } from "./adapters";
import { sortLeaderboardRows } from "./sorting";
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
    data.final_rows.length > 100000
  )
    throw new Error("不支持的滚榜数据包");
  const initial = new Set(data.initial_rows.map((r: any) => r.user_id));
  if (
    data.final_rows.some(
      (r: any) => !initial.has(r.user_id) || !Array.isArray(r.problems),
    )
  )
    throw new Error("滚榜参赛者或题目数据不完整");
  return data;
}
export function replayFrames(packet: ReplayPacket): LeaderboardData[] {
  const rows = structuredClone(packet.initial_rows);
  const hidden = new Set<string>();
  for (const row of rows)
    for (const cell of row.problems) {
      const target = packet.final_rows
        .find((r) => r.user_id === row.user_id)
        ?.problems.find((c: any) => c.problem_id === cell.problem_id);
      if (
        target &&
        cell.status !== "accepted" &&
        (cell.best_score || 0) < 100 &&
        JSON.stringify(cell) !== JSON.stringify(target)
      )
        hidden.add(`${row.user_id}:${cell.problem_id}`);
    }
  const build = () => {
    const visible = rows.map((row) => ({
      ...row,
      problems: row.problems.map((cell: any) =>
        hidden.has(`${row.user_id}:${cell.problem_id}`)
          ? { ...cell, status: "frozen" }
          : cell,
      ),
    }));
    const board = adaptTeamContestRanking({
      contest: packet.contest,
      problems: packet.problems,
      rows: visible,
      final_ready: true,
    });
    if (packet.freeze_at && packet.contest.starts_at)
      board.currentTimeSeconds = Math.max(
        0,
        Math.min(
          board.durationSeconds,
          (Date.parse(packet.freeze_at) -
            Date.parse(packet.contest.starts_at)) /
            1000,
        ),
      );
    board.replaying = true;
    board.rows = sortLeaderboardRows(
      board.rows,
      board.scoringRule,
      "performance",
    );
    let rank = 0;
    for (const row of board.rows) {
      if (!row.unranked) row.rank = ++rank;
      else row.rank = 0;
    }
    board.participantCount = rank;
    return board;
  };
  if (
    rows.length *
      (1 +
        packet.final_rows.reduce(
          (sum: number, r: any) => sum + r.problems.length,
          0,
        )) >
    500000
  )
    throw new Error("数据包过大，请按分组导出滚榜");
  const frames = [build()];
  for (const target of [...packet.final_rows].reverse()) {
    const row = rows.find((r) => r.user_id === target.user_id);
    for (const finalCell of target.problems) {
      const index = row.problems.findIndex(
        (c: any) => c.problem_id === finalCell.problem_id,
      );
      if (
        index < 0 ||
        JSON.stringify(row.problems[index]) === JSON.stringify(finalCell)
      )
        continue;
      hidden.delete(`${row.user_id}:${finalCell.problem_id}`);
      row.problems[index] = structuredClone(finalCell);
      row.solved = row.problems.filter(
        (c: any) => c.solved || c.status === "accepted" || c.best_score >= 100,
      ).length;
      row.total_score = row.problems.reduce(
        (s: number, c: any) => s + (c.best_score || 0),
        0,
      );
      row.penalty_minutes = row.problems.reduce(
        (s: number, c: any) =>
          s +
          (c.solved || c.status === "accepted"
            ? (c.elapsed_minutes || 0) + (c.wrong_attempts || 0) * 20
            : 0),
        0,
      );
      row.submission_count = row.problems.reduce(
        (s: number, c: any) => s + (c.attempts || 0),
        0,
      );
      frames.push(build());
    }
  }
  frames[frames.length - 1] = adaptTeamContestRanking({
    contest: packet.contest,
    problems: packet.problems,
    rows: packet.final_rows,
    final_ready: true,
  });
  frames[frames.length - 1].replaying = true;
  return frames;
}
