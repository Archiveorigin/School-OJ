import type { LeaderboardRow, LeaderboardScoringRule } from "./types";

export type LeaderboardViewMode = "published" | "performance";

export function sortLeaderboardRows(
  rows: readonly LeaderboardRow[],
  scoringRule: LeaderboardScoringRule,
  mode: LeaderboardViewMode,
) {
  const sorted = [...rows];
  const participation = (left: LeaderboardRow, right: LeaderboardRow) =>
    Number(right.submissions > 0) - Number(left.submissions > 0);
  if (mode === "published") return sorted.sort(participation);
  if (scoringRule === "score") {
    return sorted.sort(
      (left, right) =>
        participation(left, right) ||
        right.metric - left.metric ||
        right.solved - left.solved ||
        (left.unranked || right.unranked ? 0 : left.rank - right.rank),
    );
  }
  return sorted.sort(
    (left, right) =>
      participation(left, right) ||
      right.solved - left.solved ||
      left.metric - right.metric ||
      (left.unranked || right.unranked ? 0 : left.rank - right.rank),
  );
}
