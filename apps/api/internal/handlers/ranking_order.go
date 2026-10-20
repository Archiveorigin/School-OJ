package handlers

import "time"

// Display names and IDs stabilize display order only; they never break a
// competitive tie. Both live boards and exports use competition ranks (1,1,3).
type rankingKey struct {
	score, solved, penalty int
	effective, accepted    *time.Time
}

func compareRankingKeys(a, b rankingKey, rule string) int {
	compare := func(x, y int) int {
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
		return 0
	}
	if rule == "acm" {
		if c := compare(b.solved, a.solved); c != 0 {
			return c
		}
		if c := compare(a.penalty, b.penalty); c != 0 {
			return c
		}
		return compareTimePtr(a.accepted, b.accepted)
	}
	if c := compare(b.score, a.score); c != 0 {
		return c
	}
	if c := compare(b.solved, a.solved); c != 0 {
		return c
	}
	if rule == "ioi" {
		return compareTimePtr(a.effective, b.effective)
	}
	return 0
}

func (r examRankingRow) rankingKey() rankingKey {
	return rankingKey{r.TotalScore, r.Solved, r.PenaltyMinutes, r.LastSubmission, r.lastAccepted}
}

func (r teamContestRankingRow) rankingKey() rankingKey {
	return rankingKey{r.TotalScore, r.Solved, r.PenaltyMinutes, r.LastSubmission, r.lastAccepted}
}

func laterTime(previous, current *time.Time) *time.Time {
	if current != nil && (previous == nil || current.After(*previous)) {
		value := *current
		return &value
	}
	return previous
}
