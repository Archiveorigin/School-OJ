package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"school-oj/apps/api/internal/models"
	"testing"
	"time"
)

func TestDraftSavesDoNotCreateReviewRounds(t *testing.T) {
	for _, kind := range []string{"exam", "contest"} {
		t.Run(kind, func(t *testing.T) {
			f := newExamFixture(t)
			base, id := eventTestPaper(t, f, kind)
			for i := 0; i < 3; i++ {
				revision := eventRevision(t, f, kind, id)
				for j := 0; j < 2; j++ {
					assertStatus(t, f.request("GET", base, &f.teacher, nil), 200)
				}
				if eventRevision(t, f, kind, id) != revision {
					t.Fatal("read changed revision")
				}
				start := time.Now().Add(time.Hour)
				assertStatus(t, f.request("PUT", base+"/settings", &f.teacher, gin.H{"revision": revision, "title": "草稿", "description": fmt.Sprintf("草稿修改 %d", i), "starts_at": start, "ends_at": start.Add(time.Hour)}), 200)
				current, _ := loadEventPaper(f.s.DB, kind, id, false)
				if current.review().SubmissionVersion != 0 {
					t.Fatal("draft save counted as a review submission")
				}
				assertStatus(t, f.request("PUT", base+"/settings", &f.teacher, gin.H{"revision": revision, "title": "过期编辑", "starts_at": start, "ends_at": start.Add(time.Hour)}), 409)
			}
			assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "submit"), 200)
			current, _ := loadEventPaper(f.s.DB, kind, id, false)
			if current.review().SubmissionVersion != 1 {
				t.Fatalf("first review must be round 1: %+v", current.review())
			}
		})
	}
}

func TestFinalRankingWaitsForEverySubmission(t *testing.T) {
	for _, kind := range []string{"exam", "contest"} {
		t.Run(kind, func(t *testing.T) {
			f, base, p, start := rankingFixture(t, kind, "ioi")
			endpoint := fmt.Sprintf("/api/exams/%d/ranking", p.id)
			if kind == "contest" {
				endpoint = fmt.Sprintf("/api/contests/%d/ranking", p.id)
			}
			sub := rankingSubmission(t, f, p, f.student, models.StatusQueued, 0, start.Add(time.Minute))
			read := func(want bool) {
				t.Helper()
				response := f.request("GET", endpoint, &f.admin, nil)
				assertStatus(t, response, 200)
				var body struct {
					FinalReady bool `json:"final_ready"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.FinalReady != want || readStandings(t, f, base).FinalReady != want {
					t.Fatalf("final_ready should be %v: %s", want, response.Body.String())
				}
			}
			read(false)
			f.s.DB.Model(&sub).Updates(map[string]any{"status": models.StatusSystemError})
			read(false)
			f.s.DB.Model(&sub).Updates(map[string]any{"status": models.StatusAccepted, "score": 100})
			read(true)
			future := time.Now().Add(time.Hour)
			updates := map[string]any{"starts_at": time.Now().Add(-time.Minute)}
			if kind == "exam" {
				updates["ends_at"] = future
			}
			f.s.DB.Model(p.model()).Updates(updates)
			read(false)
		})
	}
}

func TestContestFractionalAwardsAndChoiceScope(t *testing.T) {
	f := newExamFixture(t)
	base, id := eventTestPaper(t, f, "contest")
	p, _ := loadEventPaper(f.s.DB, "contest", id, false)
	start := time.Now().Add(time.Hour)
	endpoint := fmt.Sprintf("/api/contests/%d", id)
	payload := gin.H{"revision": p.review().ReviewRevision, "title": p.contest.Title, "starts_at": start, "duration_minutes": 60, "scoring_rule": "ioi", "gold_award_percent": 16.666666, "silver_award_percent": 0, "bronze_award_percent": 16.666666}
	assertStatus(t, f.request("PUT", endpoint, &f.teacher, payload), 200)
	current, _ := loadEventPaper(f.s.DB, "contest", id, false)
	if *current.contest.GoldAwardPercent != 16.666666 || *current.contest.SilverAwardPercent != 0 {
		t.Fatalf("fractional allocation lost: %+v", current.contest)
	}
	payload["revision"] = current.review().ReviewRevision
	payload["gold_award_percent"] = 16.6666667
	assertStatus(t, f.request("PUT", endpoint, &f.teacher, payload), 400)
	own := models.Problem{OwnerID: f.teacher.ID, Slug: "choice-own", DisplayCode: "CHOICE1", Title: "own", TeamID: &p.contest.TeamID}
	outsider := models.Team{Name: "other", Slug: "choice-other", OwnerID: f.admin.ID}
	if err := f.s.DB.Create(&outsider).Error; err != nil {
		t.Fatal(err)
	}
	other := models.Problem{OwnerID: f.admin.ID, Slug: "choice-other", DisplayCode: "CHOICE2", Title: "other", TeamID: &outsider.ID}
	for _, problem := range []*models.Problem{&own, &other} {
		if err := f.s.DB.Create(problem).Error; err != nil {
			t.Fatal(err)
		}
		version := models.ProblemVersion{ProblemID: problem.ID, Version: 1, Title: problem.Title, CreatedBy: problem.OwnerID, Manifest: f.version.Manifest}
		if err := f.s.DB.Create(&version).Error; err != nil {
			t.Fatal(err)
		}
		if err := f.s.DB.Model(problem).Update("current_version_id", version.ID).Error; err != nil {
			t.Fatal(err)
		}
	}
	response := f.request("GET", base+"/choices?source=team&page=1", &f.admin, nil)
	assertStatus(t, response, 200)
	var page struct {
		Items    []models.Problem `json:"items"`
		Total    int64            `json:"total"`
		PageSize int              `json:"page_size"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != own.ID || page.Total != 1 || page.PageSize != 20 {
		t.Fatalf("wrong scoped choices: %s", response.Body.String())
	}
	assertStatus(t, f.request("GET", base+"/choices?source=team&page=0", &f.teacher, nil), 400)
	assertStatus(t, f.request("GET", base+"/choices?source=invalid", &f.teacher, nil), 400)
	revision := eventRevision(t, f, "contest", id)
	selection := []gin.H{{"problem_id": f.problem.ID, "label": "A"}, {"problem_id": own.ID, "label": "B"}, {"problem_id": other.ID, "label": "C"}}
	assertStatus(t, f.request("PUT", base+"/problems", &f.teacher, gin.H{"revision": revision, "problems": selection}), 403)
	var count int64
	f.s.DB.Model(&models.TeamContestProblem{}).Where("contest_id=?", id).Count(&count)
	if count != 1 || eventRevision(t, f, "contest", id) != revision {
		t.Fatal("failed batch partially changed the paper")
	}
	assertStatus(t, f.request("PUT", base+"/problems", &f.teacher, gin.H{"revision": revision, "problems": selection[:2]}), 200)
	if eventRevision(t, f, "contest", id) != revision+1 {
		t.Fatal("batch should change revision once")
	}
	assertStatus(t, f.request("PUT", base+"/problems", &f.teacher, gin.H{"revision": revision, "problems": selection[:2]}), 409)
}
