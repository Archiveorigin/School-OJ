import type { LeaderboardAwardPercents, LeaderboardAwardTier } from './types'

export const DEFAULT_AWARD_PERCENTS: Readonly<LeaderboardAwardPercents> = Object.freeze({
  gold: 10,
  silver: 10,
  bronze: 10
})

function percentage(value: unknown, fallback: number) {
  const number = Number(value)
  if (!Number.isFinite(number)) return fallback
  return Math.min(100, Math.max(0, Math.round(number)))
}

export function normalizeAwardPercents(value?: Partial<LeaderboardAwardPercents> | null): LeaderboardAwardPercents {
  const normalized = {
    gold: percentage(value?.gold, DEFAULT_AWARD_PERCENTS.gold),
    silver: percentage(value?.silver, DEFAULT_AWARD_PERCENTS.silver),
    bronze: percentage(value?.bronze, DEFAULT_AWARD_PERCENTS.bronze)
  }
  const total = normalized.gold + normalized.silver + normalized.bronze
  if (total <= 100) return normalized

  // Invalid server data should never make more than the complete field an award.
  const scale = 100 / total
  const gold = Math.floor(normalized.gold * scale)
  const silver = Math.floor(normalized.silver * scale)
  return { gold, silver, bronze: 100 - gold - silver }
}

export function awardCutoffs(totalParticipants: number, value?: Partial<LeaderboardAwardPercents> | null) {
  const total = Math.max(0, Math.floor(Number(totalParticipants) || 0))
  const percents = normalizeAwardPercents(value)
  const gold = Math.ceil(total * percents.gold / 100)
  const silver = Math.ceil(total * (percents.gold + percents.silver) / 100)
  const bronze = Math.ceil(total * (percents.gold + percents.silver + percents.bronze) / 100)
  return { gold, silver, bronze, percents }
}

export function awardTierForRank(
  rank: number,
  totalParticipants: number,
  value?: Partial<LeaderboardAwardPercents> | null
): LeaderboardAwardTier {
  const position = Math.floor(Number(rank) || 0)
  if (position < 1 || totalParticipants < 1) return 'none'
  const cutoffs = awardCutoffs(totalParticipants, value)
  if (cutoffs.percents.gold > 0 && position <= cutoffs.gold) return 'gold'
  if (cutoffs.percents.silver > 0 && position <= cutoffs.silver) return 'silver'
  if (cutoffs.percents.bronze > 0 && position <= cutoffs.bronze) return 'bronze'
  return 'none'
}
