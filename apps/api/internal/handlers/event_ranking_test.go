package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"school-oj/apps/api/internal/models"
)

func rankingFixture(t *testing.T, kind, rule string) (*examTestFixture, string, eventPaper, time.Time) {
	t.Helper()
	f := newExamFixture(t)
	base, id := eventTestPaper(t, f, kind)
	p, err := loadEventPaper(f.s.DB, kind, id, false)
	if err != nil {
		t.Fatal(err)
	}
	if kind == "contest" {
		if err := f.s.DB.Create(&models.TeamMembership{TeamID: p.contest.TeamID, UserID: f.peer.ID, Role: models.TeamRoleMember}).Error; err != nil {
			t.Fatal(err)
		}
	} else {
		f.s.DB.Model(&models.ExamProblem{}).Where("exam_id=?", id).Update("score", 100)
	}
	f.s.DB.Model(p.model()).Update("scoring_rule", rule)
	assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.teacher, "submit"), 200)
	assertStatus(t, eventReviewRequest(t, f, base, kind, id, &f.admin, "approve"), 200)
	start := time.Now().UTC().Truncate(time.Second).Add(-2 * time.Hour)
	updates := map[string]any{"starts_at": start}
	if kind == "exam" {
		updates["ends_at"] = start.Add(time.Hour)
	}
	if err := f.s.DB.Model(p.model()).Updates(updates).Error; err != nil {
		t.Fatal(err)
	}
	p, err = loadEventPaper(f.s.DB, kind, id, false)
	if err != nil {
		t.Fatal(err)
	}
	return f, base, p, start
}

func rankingSubmission(t *testing.T, f *examTestFixture, p eventPaper, user models.User, status models.SubmissionStatus, score int, at time.Time) models.Submission {
	t.Helper()
	sub := models.Submission{UserID: user.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Status: status, Score: score, CreatedAt: at, Language: "cpp"}
	if p.kind == "exam" {
		sub.ExamID = &p.id
	} else {
		sub.TeamContestID = &p.id
	}
	if err := f.s.DB.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	return sub
}

func readStandings(t *testing.T, f *examTestFixture, base string) eventStandings {
	t.Helper()
	response := f.request("GET", base+"/ranking", &f.admin, nil)
	assertStatus(t, response, 200)
	var data eventStandings
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestEventStandingsScoringAndExports(t *testing.T) {
	for _, kind := range []string{"exam", "contest"} {
		for _, rule := range []string{"acm", "oi", "ioi"} {
			t.Run(kind+"/"+rule, func(t *testing.T) {
				f, base, p, start := rankingFixture(t, kind, rule)
				// Alpha: CE, WA, AC, then WA. Beta: later AC. OI must
				// use the final WA, IOI the best score, ACM CE-free penalty.
				rankingSubmission(t, f, p, f.student, models.StatusCompileError, 0, start.Add(time.Minute))
				rankingSubmission(t, f, p, f.student, models.StatusWrongAnswer, 40, start.Add(2*time.Minute))
				rankingSubmission(t, f, p, f.student, models.StatusAccepted, 100, start.Add(10*time.Minute))
				rankingSubmission(t, f, p, f.student, models.StatusWrongAnswer, 20, start.Add(50*time.Minute))
				rankingSubmission(t, f, p, f.peer, models.StatusAccepted, 100, start.Add(30*time.Minute))
				// Out-of-window submissions must not change scores or ranks.
				rankingSubmission(t, f, p, f.student, models.StatusAccepted, 100, start.Add(time.Hour))
				rankingSubmission(t, f, p, f.student, models.StatusAccepted, 100, start.Add(-time.Minute))
				data := readStandings(t, f, base)
				wantFirst := f.student.ID
				if rule == "oi" {
					wantFirst = f.peer.ID
				}
				if data.Rows[0].UserID != wantFirst || data.ScoringRule != rule {
					t.Fatalf("wrong actual ranking for %s: %+v", rule, data.Rows)
				}
				var alpha examRankingRow
				for _, row := range data.Rows {
					if row.UserID == f.student.ID {
						alpha = row
					}
				}
				if alpha.SubmissionCount != 4 {
					t.Fatalf("out-of-window runs counted: %+v", alpha)
				}
				if rule == "acm" && (alpha.PenaltyMinutes != 30 || alpha.Problems[0].WrongAttempts != 1) {
					t.Fatalf("CE affected ACM penalty: %+v", alpha)
				}
				if rule == "oi" && (alpha.TotalScore != 20 || alpha.Solved != 0) {
					t.Fatalf("OI reused historical AC: %+v", alpha)
				}
				if rule == "oi" && !data.Rows[0].Problems[0].Fastest {
					t.Fatal("OI fastest marker must use the effective final result")
				}
				for _, format := range []string{"xlsx", "html"} {
					response := f.request("GET", base+"/ranking/export?format="+format, &f.admin, nil)
					assertStatus(t, response, 200)
					if response.Header().Get("Cache-Control") != "private, no-store" {
						t.Fatal("export must not be cached")
					}
					if format == "html" {
						if !strings.Contains(response.Body.String(), "待审核") {
							t.Fatal("missing state/scoring metadata")
						}
					} else {
						files := unzipRanking(t, response.Body.Bytes())
						if !strings.Contains(files["xl/worksheets/sheet1.xml"], "待审核") || files["xl/styles.xml"] == "" {
							t.Fatal("missing styled ranking sheet")
						}
					}
				}
				if kind == "exam" {
					for _, format := range []string{"xlsx", "markdown"} {
						response := f.request("GET", fmt.Sprintf("/api/exams/%d/report/export?format=%s", p.id, format), &f.teacher, nil)
						assertStatus(t, response, 200)
						if format == "markdown" && !strings.Contains(response.Body.String(), "- 赛制："+strings.ToUpper(rule)) {
							t.Fatal("legacy export ignored actual scoring rule")
						}
					}
				}
				assertStatus(t, f.request("GET", base+"/ranking", &f.student, nil), 403)
				assertStatus(t, f.request("GET", base+"/ranking/export?format=html", &f.student, nil), 403)
				assertStatus(t, f.request("GET", base+"/ranking/export?format=csv", &f.admin, nil), 400)
				// Late judge results must appear after the deadline; export is
				// withheld until the result is settled.
				queued := rankingSubmission(t, f, p, f.peer, models.StatusQueued, 0, start.Add(55*time.Minute))
				if readStandings(t, f, base).Checks.Judging != 1 {
					t.Fatal("pending judge result not reported")
				}
				assertStatus(t, f.request("GET", base+"/ranking/export", &f.admin, nil), 409)
				f.s.DB.Model(&queued).Update("status", models.StatusSystemError)
				if readStandings(t, f, base).Checks.Failed != 1 {
					t.Fatal("judge errors must be visible to reviewers")
				}
				assertStatus(t, f.request("GET", base+"/ranking/export", &f.admin, nil), 409)
				assertStatus(t, eventReviewRequest(t, f, base, kind, p.id, &f.admin, "complete"), 409)
				f.s.DB.Model(&queued).Updates(map[string]any{"status": models.StatusWrongAnswer, "score": 10})
				updated := readStandings(t, f, base)
				if updated.Checks.Judging != 0 {
					t.Fatal("finished judge result not refreshed")
				}
				if rule == "oi" && updated.Rows[0].UserID != f.student.ID {
					t.Fatal("late OI result did not reorder actual ranking")
				}
				assertStatus(t, eventReviewRequest(t, f, base, kind, p.id, &f.admin, "complete"), 200)
				final := f.request("GET", base+"/ranking/export?format=html", &f.admin, nil)
				assertStatus(t, final, 200)
				if !strings.Contains(final.Body.String(), "已确认") {
					t.Fatal("final export not distinguished from review copy")
				}
				f.s.DB.Model(p.model()).Update("deleted_at", time.Now())
				assertStatus(t, f.request("GET", base+"/ranking/export", &f.admin, nil), 404)
			})
		}
	}
}

func TestEventRankingTiesAndManualReview(t *testing.T) {
	for _, kind := range []string{"exam", "contest"} {
		t.Run(kind, func(t *testing.T) {
			f, base, p, start := rankingFixture(t, kind, "acm")
			rankingSubmission(t, f, p, f.student, models.StatusAccepted, 100, start.Add(10*time.Minute))
			rankingSubmission(t, f, p, f.peer, models.StatusAccepted, 100, start.Add(10*time.Minute))
			data := readStandings(t, f, base)
			if data.Rows[0].Rank != 1 || data.Rows[1].Rank != 1 {
				t.Fatalf("equal results must tie: %+v", data.Rows)
			}
			if kind == "contest" && data.Rows[2].Rank != 3 {
				t.Fatalf("competition ranking should skip tied positions: %+v", data.Rows)
			}
			if kind == "exam" {
				f.s.DB.Model(p.model()).Update("settings", map[string]any{"manual_review": true})
				if readStandings(t, f, base).FinalReady {
					t.Fatal("ungraded manual submissions must prevent final ranking")
				}
				assertStatus(t, f.request("GET", base+"/ranking/export", &f.admin, nil), 409)
			}
		})
	}
}

func unzipRanking(t *testing.T, body []byte) map[string]string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, file := range reader.File {
		r, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, _ := io.ReadAll(r)
		r.Close()
		files[file.Name] = string(content)
	}
	return files
}

func TestRankingExportEscapesNamesAndPreservesIDs(t *testing.T) {
	data := eventStandings{
		Kind: "exam", Title: `<img src=x onerror=alert(1)>`, ScoringRule: "acm", Lifecycle: "closed",
		Rows: []examRankingRow{{Rank: 1, UserID: 7, Name: "=HYPERLINK(\"http://bad\")", studentNo: "000123"}},
	}
	body, err := renderStandingsHTML(data)
	if err != nil || strings.Contains(string(body), "<img src=x") {
		t.Fatalf("unescaped export: %s %v", body, err)
	}
	body, err = renderStandingsXLSX(data)
	if err != nil {
		t.Fatal(err)
	}
	sheet := unzipRanking(t, body)["xl/worksheets/sheet1.xml"]
	if strings.Contains(sheet, "<f>") || !strings.Contains(sheet, "000123") || !strings.Contains(sheet, `state="frozen"`) {
		t.Fatal("unsafe names, lost ID formatting, or missing frozen headers")
	}
}
