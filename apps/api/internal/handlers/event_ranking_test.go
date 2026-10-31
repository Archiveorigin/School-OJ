package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	requireOfflineRanking(t)
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
						if !strings.Contains(response.Body.String(), `"lifecycle":"closed"`) {
							t.Fatal("missing state/scoring metadata")
						}
					} else {
						files := unzipRanking(t, response.Body.Bytes())
						if !strings.Contains(files["xl/worksheets/sheet1.xml"], "通过题数") || files["xl/styles.xml"] == "" {
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
				if !strings.Contains(final.Body.String(), `"lifecycle":"completed"`) {
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

func TestRankingExportEscapesNamesAndKeepsOnlySummaryColumns(t *testing.T) {
	requireOfflineRanking(t)
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
	if strings.Contains(sheet, "<f>") || strings.Contains(sheet, "000123") || strings.Contains(sheet, "E1") || !strings.Contains(sheet, `state="frozen"`) {
		t.Fatal("unsafe names, unexpected columns, or missing frozen headers")
	}
}

func TestRankingExportTitleAndFinalTimeline(t *testing.T) {
	requireOfflineRanking(t)
	start := time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	data := eventStandings{Title: "秋季程序设计赛", Kind: "contest", ScoringRule: "acm", Lifecycle: "completed", StartsAt: &start, EndsAt: &end}
	body, err := renderStandingsHTML(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{`"title":"秋季程序设计赛"`, `"starts_at":"2026-10-31T09:00:00Z"`, `"ends_at":"2026-10-31T11:00:00Z"`, "scoreboard-core", `"lifecycle":"completed"`} {
		if !strings.Contains(string(body), text) {
			t.Fatalf("missing %s", text)
		}
	}
	if rankingExportName("秋季程序设计赛", "xlsx") != "秋季程序设计赛.xlsx" {
		t.Fatal("title filename lost")
	}
	if strings.ContainsAny(rankingExportDisposition("坏\r\n标题/测试", "html"), "\r\n") {
		t.Fatal("unsafe disposition")
	}
}

func TestRankingExportOfflineFixture(t *testing.T) {
	requireOfflineRanking(t)
	start := time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	data := eventStandings{Kind: "contest", Title: "秋季程序设计赛", ScoringRule: "acm", Lifecycle: "completed", StartsAt: &start, EndsAt: &end,
		Problems: []examRankingProblem{{ProblemID: 1, Label: "A", Title: "求和", Score: 100}},
		Rows: []examRankingRow{
			{UserID: 1, Rank: 0, Unranked: true, Name: "打星选手", Solved: 1, PenaltyMinutes: 12, SubmissionCount: 1, Problems: []examRankingCell{{ProblemID: 1, Status: "accepted", Attempts: 1, ElapsedMinutes: 12, Fastest: true}}},
			{UserID: 2, Rank: 1, Name: "=公式安全", Solved: 1, PenaltyMinutes: 40, SubmissionCount: 2, Problems: []examRankingCell{{ProblemID: 1, Status: "accepted", Attempts: 2, ElapsedMinutes: 20, WrongAttempts: 1}}},
			{UserID: 3, Rank: 2, Name: "普通选手", SubmissionCount: 1, Problems: []examRankingCell{{ProblemID: 1, Status: "wrong_answer", Attempts: 1}}},
			{UserID: 4, Rank: 2, Name: "未提交", Problems: []examRankingCell{{ProblemID: 1}}},
		}}
	html, err := renderStandingsHTML(data)
	if err != nil {
		t.Fatal(err)
	}
	xlsx, err := renderStandingsXLSX(data)
	if err != nil {
		t.Fatal(err)
	}
	sheet := unzipRanking(t, xlsx)["xl/worksheets/sheet1.xml"]
	for _, text := range []string{"名次", "参赛者", "通过题数", "罚时 / 分钟"} {
		if !strings.Contains(sheet, text) {
			t.Fatal(text)
		}
	}
	if strings.Contains(sheet, `r="E`) || strings.Contains(sheet, "<f>") {
		t.Fatal("unexpected extra data or formula")
	}
	if dir := os.Getenv("OJ_EXPORT_QA_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "standings.html"), html, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "standings.xlsx"), xlsx, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func requireOfflineRanking(t *testing.T) {
	t.Helper()
	if _, err := offlineRankingTemplate(); err != nil {
		t.Skip("build offline runtime first: cd apps/web && npm run build:ranking-export")
	}
}

func TestFirstBloodHasOneWinnerAcrossStarredAndOfficial(t *testing.T) {
	start := time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC)
	at := start.Add(time.Minute)
	users := []models.User{{ID: 2, Name: "官方"}, {ID: 1, Name: "打星"}}
	links := []models.TeamContestProblem{{ProblemID: 11}}
	aggregates := []teamContestCellAggregate{
		{UserID: 2, ProblemID: 11, SubmissionCount: 1, Attempts: 1, SolvedAt: &at, LatestStatus: "accepted", BestScore: 100, LastSubmission: at},
		{UserID: 1, ProblemID: 11, SubmissionCount: 1, Attempts: 1, SolvedAt: &at, LatestStatus: "accepted", BestScore: 100, LastSubmission: at},
	}
	rows := buildTeamContestRanking(users, links, aggregates, start, "acm")
	contest := models.TeamContest{}
	contest.RankingUserIDs = []byte(`[2]`)
	if err := markContestRankingRows(rows, contest); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, row := range rows {
		if row.Problems[0].Fastest {
			count++
			if row.UserID != 1 || !row.Unranked {
				t.Fatal("wrong winner")
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected one winner, got %d", count)
	}
	examRows := []examRankingRow{{UserID: 2, Problems: []examRankingCell{{ProblemID: 11, firstAccepted: &at, Fastest: true}}}, {UserID: 1, Problems: []examRankingCell{{ProblemID: 11, firstAccepted: &at}}}}
	markExamFastestCells(examRows)
	if examRows[0].Problems[0].Fastest || !examRows[1].Problems[0].Fastest {
		t.Fatal("exam tie has multiple winners")
	}
}
