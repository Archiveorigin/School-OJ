import { describe, expect, it } from 'vitest'
import { awardCutoffs, awardTierForRank, normalizeAwardPercents } from '../src/features/leaderboard/awards'

describe('leaderboard award ranges', () => {
  it('defaults to cumulative gold, silver and bronze ranges covering the top 30%', () => {
    expect(awardCutoffs(100)).toMatchObject({ gold: 10, silver: 20, bronze: 30 })
    expect(awardTierForRank(10, 100)).toBe('gold')
    expect(awardTierForRank(11, 100)).toBe('silver')
    expect(awardTierForRank(21, 100)).toBe('bronze')
    expect(awardTierForRank(31, 100)).toBe('none')
  })

  it('uses ceilings for small participant groups and supports zero-sized tiers', () => {
    expect(awardCutoffs(7)).toMatchObject({ gold: 1, silver: 2, bronze: 3 })
    expect(awardTierForRank(1, 7, { gold: 0, silver: 20, bronze: 10 })).toBe('silver')
  })

  it('normalizes malformed values into a bounded allocation', () => {
    const normalized = normalizeAwardPercents({ gold: 80, silver: 80, bronze: 80 })
    expect(normalized.gold + normalized.silver + normalized.bronze).toBe(100)
    expect(normalizeAwardPercents({ gold: -2, silver: Number.NaN, bronze: 12.4 })).toEqual({ gold: 0, silver: 10, bronze: 12 })
  })
})
