export interface ContestAwardPercents {
  gold_award_percent: number
  silver_award_percent: number
  bronze_award_percent: number
}

export const defaultContestAwardPercents = (): ContestAwardPercents => ({
  gold_award_percent: 10,
  silver_award_percent: 10,
  bronze_award_percent: 10,
})

export function contestAwardTotal(value: ContestAwardPercents) {
  return value.gold_award_percent + value.silver_award_percent + value.bronze_award_percent
}

export function contestAwardValidationError(value: ContestAwardPercents) {
  const percents = [value.gold_award_percent, value.silver_award_percent, value.bronze_award_percent]
  if (percents.some((percent) => !Number.isInteger(percent) || percent < 0 || percent > 100)) {
    return '金、银、铜奖比例必须是 0 到 100 之间的整数'
  }
  if (contestAwardTotal(value) > 100) return '金、银、铜奖比例之和不能超过 100%'
  return ''
}
