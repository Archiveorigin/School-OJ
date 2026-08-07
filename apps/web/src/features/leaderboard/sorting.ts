import type { LeaderboardRow, LeaderboardScoringRule } from './types'

export type LeaderboardViewMode = 'published' | 'performance'

export function sortLeaderboardRows(
  rows: readonly LeaderboardRow[],
  scoringRule: LeaderboardScoringRule,
  mode: LeaderboardViewMode
) {
  const sorted = [...rows]
  if (mode === 'published') return sorted.sort((left, right) => left.rank - right.rank)
  if (scoringRule === 'score') {
    return sorted.sort((left, right) =>
      right.metric - left.metric ||
      right.solved - left.solved ||
      left.rank - right.rank
    )
  }
  return sorted.sort((left, right) =>
    right.solved - left.solved ||
    left.metric - right.metric ||
    left.rank - right.rank
  )
}
