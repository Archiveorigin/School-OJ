import type { LeaderboardRow, LeaderboardScoringRule } from "./types";

export type LeaderboardViewMode = "published" | "performance";

export function sortLeaderboardRows(
  rows: readonly LeaderboardRow[],
  scoringRule: LeaderboardScoringRule,
  mode: LeaderboardViewMode,
) {
  const sorted = [...rows];
  if (mode === "published") return sorted;
  if (scoringRule === "score") {
    return sorted.sort(
      (left, right) =>
        right.metric - left.metric ||
        right.solved - left.solved ||
        (left.unranked || right.unranked ? 0 : left.rank - right.rank),
    );
  }
  return sorted.sort(
    (left, right) =>
      right.solved - left.solved ||
      left.metric - right.metric ||
      (left.unranked || right.unranked ? 0 : left.rank - right.rank),
  );
}
