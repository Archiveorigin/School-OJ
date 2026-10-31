package handlers

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"school-oj/apps/api/internal/models"
)

func TestContestRankingRosterDoesNotRestrictParticipation(t *testing.T) {
	f := newExamFixture(t)
	base, id := eventTestPaper(t, f, "contest")
	p, _ := loadEventPaper(f.s.DB, "contest", id, false)
	endpoint := fmt.Sprintf("/api/contests/%d", id)
	payload := gin.H{"title": "名单测试", "starts_at": time.Now().Add(time.Hour), "duration_minutes": 60, "scoring_rule": "ioi", "revision": p.review().ReviewRevision, "ranking_user_ids": []uint{f.teacher.ID, f.teacher.ID}}
	assertStatus(t, f.request("PUT", endpoint, &f.teacher, payload), 200)
	p, _ = loadEventPaper(f.s.DB, "contest", id, false)
	ids, selected, err := contestRankingIDs(p.contest.RankingUserIDs)
	if err != nil || !selected || len(ids) != 1 || ids[0] != f.teacher.ID {
		t.Fatalf("roster not persisted/deduplicated: %s", p.contest.RankingUserIDs)
	}
	_, ranked, err := f.s.loadContestRankingParticipants(p.contest)
	if err != nil || len(ranked) != 1 || ranked[0].ID != f.teacher.ID {
		t.Fatalf("wrong draft ranking roster: %v %+v", err, ranked)
	}
	assertStatus(t, f.request("PUT", endpoint, &f.teacher, payload), 409)
	payload["revision"] = p.review().ReviewRevision
	payload["ranking_user_ids"] = []uint{f.peer.ID}
	assertStatus(t, f.request("PUT", endpoint, &f.teacher, payload), 400)
	payload["ranking_user_ids"] = []uint{f.student.ID}
	assertStatus(t, f.request("PUT", endpoint, &f.peer, payload), 403)
	for _, value := range []any{"bad", []int{-1}, []float64{1.5}, []uint{0}} {
		payload["ranking_user_ids"] = value
		assertStatus(t, f.request("PUT", endpoint, &f.teacher, payload), 400)
	}
	// An older client omitting the field must preserve the selected roster.
	delete(payload, "ranking_user_ids")
	assertStatus(t, f.request("PUT", endpoint, &f.teacher, payload), 200)
	p, _ = loadEventPaper(f.s.DB, "contest", id, false)
	if string(p.contest.RankingUserIDs) != fmt.Sprintf("[%d]", f.teacher.ID) {
		t.Fatal("omitted roster was reset")
	}
	assertStatus(t, eventReviewRequest(t, f, base, "contest", id, &f.teacher, "submit"), 200)
	assertStatus(t, eventReviewRequest(t, f, base, "contest", id, &f.admin, "approve"), 200)
	if err := f.s.DB.Model(&models.TeamContest{}).Where("id=?", id).Update("starts_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	// A student excluded from ranking is still allowed to submit.
	response := f.request("POST", endpoint+"/submissions", &f.student, gin.H{"problem_id": f.problem.ID, "language": "cpp", "source_code": "int main(){}"})
	assertStatus(t, response, 201)
	response = f.request("GET", endpoint+"/ranking", &f.student, nil)
	assertStatus(t, response, 200)
	var body struct {
		Rows []struct {
			UserID   uint `json:"user_id"`
			Unranked bool `json:"unranked"`
			Rank     int  `json:"rank"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Rows) != 2 || body.Rows[0].UserID != f.teacher.ID || body.Rows[0].Unranked || !body.Rows[1].Unranked || body.Rows[1].Rank != 0 {
		t.Fatalf("excluded submitter ranked: %s", response.Body.String())
	}
	p, _ = loadEventPaper(f.s.DB, "contest", id, false)
	standings, err := f.s.buildEventStandings(p, time.Now())
	if err != nil || len(standings.Rows) != 2 || standings.Rows[0].UserID != f.teacher.ID || !standings.Rows[1].Unranked || standings.Rows[1].Rank != 0 {
		t.Fatalf("export roster differs: %v", err)
	}
	payload["revision"] = p.review().ReviewRevision
	payload["ranking_user_ids"] = nil
	assertStatus(t, f.request("PUT", endpoint, &f.teacher, payload), 409)
}

func TestContestRankingRosterAllAndEmpty(t *testing.T) {
	f := newExamFixture(t)
	_, id := eventTestPaper(t, f, "contest")
	p, _ := loadEventPaper(f.s.DB, "contest", id, false)
	for _, tc := range []struct {
		raw  string
		want int
	}{{"null", 2}, {"[]", 0}} {
		roster, err := normalizeContestRankingRoster(f.s.DB, p.contest.TeamID, []byte(tc.raw))
		if err != nil {
			t.Fatal(err)
		}
		p.contest.RankingUserIDs = roster
		_, users, err := f.s.loadContestRankingParticipants(p.contest)
		if err != nil || len(users) != tc.want {
			t.Fatalf("%s: got %d: %v", tc.raw, len(users), err)
		}
	}
}

func TestStarredRankingPreservesPerformanceOrder(t *testing.T) {
	rows := []teamContestRankingRow{
		{UserID: 1, Rank: 1}, {UserID: 2, Rank: 2},
		{UserID: 3, Rank: 2}, {UserID: 4, Rank: 4},
	}
	contest := models.TeamContest{}
	contest.RankingUserIDs = []byte(`[2,3,4]`)
	if err := markContestRankingRows(rows, contest); err != nil {
		t.Fatal(err)
	}
	for i, row := range rows {
		if row.UserID != uint(i+1) {
			t.Fatalf("performance order changed: %+v", rows)
		}
	}
	if !rows[0].Unranked || rows[0].Rank != 0 || rows[1].Rank != 1 || rows[2].Rank != 1 || rows[3].Rank != 3 {
		t.Fatalf("incorrect official ranks: %+v", rows)
	}
}
