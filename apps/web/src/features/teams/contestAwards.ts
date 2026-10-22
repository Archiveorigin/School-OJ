export interface ContestAwardPercents {
  gold_award_percent: number;
  silver_award_percent: number;
  bronze_award_percent: number;
}

export const defaultContestAwardPercents = (): ContestAwardPercents => ({
  gold_award_percent: 10,
  silver_award_percent: 10,
  bronze_award_percent: 10,
});

export function contestAwardTotal(value: ContestAwardPercents) {
  return (
    Math.round(
      (value.gold_award_percent +
        value.silver_award_percent +
        value.bronze_award_percent) *
        1e6,
    ) / 1e6
  );
}

export function contestAwardValidationError(value: ContestAwardPercents) {
  const percents = [
    value.gold_award_percent,
    value.silver_award_percent,
    value.bronze_award_percent,
  ];
  if (
    percents.some(
      (percent) => !Number.isFinite(percent) || percent < 0 || percent > 100,
    )
  ) {
    return "金、银、铜奖比例必须在 0 到 100 之间";
  }
  if (contestAwardTotal(value) > 100)
    return "金、银、铜奖比例之和不能超过 100%";
  return "";
}

// Round down so an input such as 1 of 6 does not become two awards through
// ceil(6 * 16.666667 / 100). The server stores six decimal places.
export function awardCountToPercent(count: number, participants: number) {
  if (
    !Number.isInteger(participants) ||
    participants < 1 ||
    !Number.isInteger(count) ||
    count < 0 ||
    count > participants
  )
    return 0;
  return Math.floor((count * 100_000_000) / participants) / 1_000_000;
}

export function awardCountsFromPercents(
  value: ContestAwardPercents,
  participants: number,
) {
  const total = Math.max(0, Math.floor(participants || 0));
  const gold = Math.ceil((total * value.gold_award_percent) / 100 - 1e-9);
  const silver = Math.ceil(
    (total * (value.gold_award_percent + value.silver_award_percent)) / 100 -
      1e-9,
  );
  const bronze = Math.ceil((total * contestAwardTotal(value)) / 100 - 1e-9);
  return [gold, silver - gold, bronze - silver].map((n) => Math.max(0, n));
}
