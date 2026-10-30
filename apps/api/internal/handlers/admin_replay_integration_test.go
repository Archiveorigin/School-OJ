package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"school-oj/apps/api/internal/models"
	"testing"
	"time"
)

func TestAdminReplayAndCompletionFreeze(t *testing.T) {
	f, base, p, start := rankingFixture(t, "contest", "acm")
	f.s.DB.Model(p.model()).Updates(map[string]any{"freeze_enabled": true, "freeze_duration_minutes": 30})
	sub := rankingSubmission(t, f, p, f.student, models.StatusAccepted, 100, start.Add(45*time.Minute))
	if f.s.shouldRedactSubmission(f.student, sub, time.Now()) || !f.s.shouldRedactSubmission(f.teacher, sub, time.Now()) {
		t.Fatal("frozen result must be visible to its author and hidden from other non-admin viewers")
	}
	endpoint := fmt.Sprintf("/api/contests/%d/ranking", p.id)
	for _, user := range []models.User{f.student, f.teacher, f.admin} {
		response := f.request("GET", endpoint, &user, nil)
		assertStatus(t, response, 200)
		var data struct {
			Frozen     bool                    `json:"frozen"`
			FinalReady bool                    `json:"final_ready"`
			Rows       []teamContestRankingRow `json:"rows"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		admin := user.ID == f.admin.ID
		if data.Frozen == admin || data.FinalReady != admin {
			t.Fatalf("incorrect freeze/final readiness for %s: %+v", user.Role, data)
		}
		for _, row := range data.Rows {
			if row.UserID == f.student.ID && (row.Solved == 1) != admin {
				t.Fatal("hidden final score disclosed", user.Role)
			}
		}
	}
	for _, user := range []models.User{f.student, f.teacher} {
		assertStatus(t, f.request("GET", base+"/ranking/replay", &user, nil), 403)
		assertStatus(t, f.request("GET", base+"/ranking/export?format=json", &user, nil), 403)
		assertStatus(t, f.request("GET", base+"/ranking/export?format=html", &user, nil), 403)
		assertStatus(t, f.request("GET", base+"/ranking", &user, nil), 403)
	}
	assertStatus(t, f.request("GET", base+"/ranking/replay", &f.admin, nil), 200)
	// Late judging is required even after the deadline. Never export stale finals.
	f.s.DB.Model(&sub).Updates(map[string]any{"status": models.StatusRunning, "score": 0})
	assertStatus(t, f.request("GET", base+"/ranking/replay", &f.admin, nil), 409)
	assertStatus(t, f.request("GET", base+"/ranking/export?format=html", &f.admin, nil), 409)
	f.s.DB.Model(&sub).Updates(map[string]any{"status": models.StatusAccepted, "score": 100})
	f.s.DB.Model(p.model()).Update("ranking_user_ids", fmt.Sprintf("[%d]", f.peer.ID))
	response := f.request("GET", base+"/ranking/export?format=html", &f.admin, nil)
	assertStatus(t, response, 200)
	if !bytes.Contains(response.Body.Bytes(), []byte("*")) || !bytes.Contains(response.Body.Bytes(), []byte("accepted")) {
		t.Fatal("HTML missing final accepted result/unranked participant")
	}
	assertStatus(t, eventReviewRequest(t, f, base, "contest", p.id, &f.admin, "complete"), 200)
	if f.s.shouldRedactSubmission(f.student, sub, time.Now()) {
		t.Fatal("completed results remained redacted")
	}
	response = f.request("GET", endpoint, &f.student, nil)
	assertStatus(t, response, 200)
	var data struct {
		Frozen     bool                    `json:"frozen"`
		FinalReady bool                    `json:"final_ready"`
		Rows       []teamContestRankingRow `json:"rows"`
	}
	json.Unmarshal(response.Body.Bytes(), &data)
	if data.Frozen || !data.FinalReady {
		t.Fatal("completion did not release public final")
	}
	found := false
	for _, row := range data.Rows {
		if row.UserID == f.student.ID && row.Solved == 1 && row.Unranked {
			found = true
		}
	}
	if !found {
		t.Fatal("released standings lost actual final score")
	}
}

func TestSubmissionAliasSharedRateLimit(t *testing.T) {
	f := newExamFixture(t)
	if f.s.Redis == nil {
		t.Skip("requires isolated Redis")
	}
	_, id := eventTestPaper(t, f, "contest")
	// A unique account keeps the rate-limit test separate from other schemas.
	user := models.User{ID: uint(time.Now().UnixNano()%1000000000) + 10000, Role: models.RoleStudent, Email: fmt.Sprintf("rate-%d@example.test", time.Now().UnixNano()), EmailVerified: true}
	if err := f.s.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		response := f.request("POST", fmt.Sprintf("/api/contests/%d/submissions", id), &user, map[string]any{})
		if response.Code == 429 {
			t.Fatal("fresh per-user budget unexpectedly exhausted", i)
		}
	}
	for _, endpoint := range []string{"/api/submissions", fmt.Sprintf("/api/contests/%d/submissions", id), "/api/problem-sets/1/submissions", "/api/teams/1/problem-sets/1/submissions", fmt.Sprintf("/api/teams/1/contests/%d/submissions", id)} {
		response := f.request("POST", endpoint, &user, map[string]any{})
		assertStatus(t, response, 429)
		if response.Header().Get("Retry-After") == "" {
			t.Fatal("missing retry guidance")
		}
	}
}
