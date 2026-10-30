package handlers

import (
	"encoding/json"
	"fmt"
	"school-oj/apps/api/internal/models"
	"strings"
	"testing"
	"time"
)

func TestContestFreezeOwnResultsRemainVisible(t *testing.T) {
	for _, rule := range []string{"acm", "ioi"} {
		t.Run(rule, func(t *testing.T) {
			f, _, p, _ := rankingFixture(t, "contest", rule)
			sub := rankingSubmission(t, f, p, f.student, models.StatusAccepted, 100, time.Now())
			for _, phase := range []string{"running", "closed"} {
				start := time.Now().Add(-45 * time.Minute)
				if phase == "closed" {
					start = time.Now().Add(-2 * time.Hour)
				}
				if err := f.s.DB.Model(p.model()).Updates(map[string]any{"starts_at": start, "freeze_enabled": true, "freeze_duration_minutes": 30}).Error; err != nil {
					t.Fatal(err)
				}
				for _, status := range []models.SubmissionStatus{models.StatusAccepted, models.StatusWrongAnswer} {
					if err := f.s.DB.Model(&sub).Updates(map[string]any{"created_at": start.Add(40 * time.Minute), "status": status, "score": 100, "time_ms": 37, "memory_kb": 4096, "message": "own-private-result"}).Error; err != nil {
						t.Fatal(err)
					}
					for _, prefix := range []string{"/api", "/api/v1"} {
						endpoints := []string{fmt.Sprintf("/submissions/%d", sub.ID), fmt.Sprintf("/submissions?team_contest_id=%d", p.id), fmt.Sprintf("/submissions/latest?team_contest_id=%d", p.id), fmt.Sprintf("/contests/%d/submissions", p.id)}
						if f.s.Redis != nil {
							endpoints = append(endpoints, fmt.Sprintf("/submissions/%d/events", sub.ID))
						}
						for _, endpoint := range endpoints {
							response := f.request("GET", prefix+endpoint, &f.student, nil)
							assertStatus(t, response, 200)
							body := response.Body.String()
							if !strings.Contains(body, fmt.Sprintf(`"status":"%s"`, status)) || !strings.Contains(body, `"time_ms":37`) || strings.Contains(body, `"status":"recorded"`) {
								t.Fatalf("%s/%s own result masked at %s: %s", rule, phase, endpoint, body)
							}
						}
						other := f.request("GET", fmt.Sprintf("%s/submissions/%d", prefix, sub.ID), &f.teacher, nil)
						assertStatus(t, other, 200)
						if !strings.Contains(other.Body.String(), `"status":"recorded"`) || strings.Contains(other.Body.String(), "own-private-result") || strings.Contains(other.Body.String(), `"time_ms":37`) {
							t.Fatal("other viewer received the private frozen result", other.Body.String())
						}
					}
					// Seeing one's own feedback must not thaw the public scoreboard.
					response := f.request("GET", fmt.Sprintf("/api/contests/%d/ranking", p.id), &f.student, nil)
					assertStatus(t, response, 200)
					var ranking struct {
						Frozen bool                    `json:"frozen"`
						Rows   []teamContestRankingRow `json:"rows"`
					}
					if err := json.Unmarshal(response.Body.Bytes(), &ranking); err != nil {
						t.Fatal(err)
					}
					if !ranking.Frozen {
						t.Fatal("private result visibility released the scoreboard")
					}
					for _, row := range ranking.Rows {
						if row.UserID == f.student.ID && (row.Solved != 0 || row.TotalScore != 0) {
							t.Fatal("post-freeze score leaked into public ranking")
						}
					}
				}
			}
		})
	}
}

func TestContestOwnFeedbackPreservesOIEmbargo(t *testing.T) {
	f, _, p, _ := rankingFixture(t, "contest", "oi")
	// A regular participant must not inherit the fixture team's authoring grant.
	if err := f.s.DB.Model(&models.Team{}).Where("id=?", p.contest.TeamID).Update("contest_permission", "owner").Error; err != nil {
		t.Fatal(err)
	}
	start := time.Now().Add(-time.Minute)
	f.s.DB.Model(p.model()).Update("starts_at", start)
	sub := rankingSubmission(t, f, p, f.student, models.StatusAccepted, 100, start.Add(time.Second))
	if !f.s.shouldRedactSubmission(f.student, sub, time.Now()) {
		t.Fatal("OI result embargo was unintentionally removed")
	}
}
