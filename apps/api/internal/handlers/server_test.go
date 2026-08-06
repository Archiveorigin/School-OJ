package handlers

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"

	"gorm.io/datatypes"
)

func TestRouterBuilds(t *testing.T) {
	_ = (Server{}).Router()
}

func TestSortExamRankingRows(t *testing.T) {
	base := time.Date(2026, 6, 7, 10, 0, 0, 0, time.UTC)
	later := base.Add(time.Minute)
	finished := base.Add(2 * time.Minute)
	rows := []examRankingRow{
		{Name: "Charlie", StudentNo: "S3", TotalScore: 80, Solved: 2, LastSubmission: &later},
		{Name: "Alice", StudentNo: "S1", TotalScore: 100, Solved: 1, LastSubmission: &later},
		{Name: "Bob", StudentNo: "S2", TotalScore: 100, Solved: 2, LastSubmission: &later, FinishedAt: &finished},
		{Name: "Ada", StudentNo: "S0", TotalScore: 100, Solved: 2, LastSubmission: &base},
	}

	sortExamRankingRows(rows)

	got := []string{rows[0].Name, rows[1].Name, rows[2].Name, rows[3].Name}
	want := []string{"Ada", "Bob", "Alice", "Charlie"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rank %d = %s, want %s; full order=%v", i+1, got[i], want[i], got)
		}
	}
}

func TestStudentExamEntryDecisionBlocksNotStartedExam(t *testing.T) {
	now := time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)
	start := now.Add(time.Hour)
	end := now.Add(2 * time.Hour)

	reason, recordAttempt := studentExamEntryDecision(models.Exam{StartsAt: &start, EndsAt: &end}, now, nil)
	if reason != "exam has not started" {
		t.Fatalf("reason = %q, want exam has not started", reason)
	}
	if recordAttempt {
		t.Fatal("not-started exam must not record an attempt")
	}
}

func TestStudentExamEntryDecisionRecordsStartedExam(t *testing.T) {
	now := time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)
	start := now.Add(-time.Minute)
	end := now.Add(time.Hour)

	reason, recordAttempt := studentExamEntryDecision(models.Exam{StartsAt: &start, EndsAt: &end}, now, nil)
	if reason != "" {
		t.Fatalf("reason = %q, want empty", reason)
	}
	if !recordAttempt {
		t.Fatal("started exam should record an attempt")
	}
}

func TestExamRankingVisibleReadsBooleanAndStringSettings(t *testing.T) {
	tests := []struct {
		name     string
		exam     models.Exam
		expected bool
	}{
		{name: "missing", exam: models.Exam{}, expected: false},
		{name: "boolean true", exam: models.Exam{Settings: map[string]interface{}{"ranking_visible": true}}, expected: true},
		{name: "boolean false", exam: models.Exam{Settings: map[string]interface{}{"ranking_visible": false}}, expected: false},
		{name: "string true", exam: models.Exam{Settings: map[string]interface{}{"ranking_visible": "true"}}, expected: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := examRankingVisible(test.exam); actual != test.expected {
				t.Fatalf("examRankingVisible() = %v, want %v", actual, test.expected)
			}
		})
	}
}

func TestProblemScoreFromSubmissionsUsesBestCompletedSubmission(t *testing.T) {
	base := time.Date(2026, 6, 13, 9, 0, 0, 0, time.UTC)
	later := base.Add(time.Minute)
	view, submitted, pending := problemScoreFromSubmissions(models.Problem{ID: 7, Title: "A+B"}, 20, false, []models.Submission{
		{ID: 3, ProblemID: 7, Status: models.StatusRunning, Score: 0, CreatedAt: later},
		{ID: 2, ProblemID: 7, Status: models.StatusAccepted, Score: 90, CreatedAt: base},
		{ID: 1, ProblemID: 7, Status: models.StatusWrongAnswer, Score: 30, CreatedAt: base.Add(-time.Minute)},
	})
	if !submitted {
		t.Fatal("expected submitted")
	}
	if !pending {
		t.Fatal("running submission should keep score pending")
	}
	if !view.ScoreReady {
		t.Fatal("expected score ready from completed submission")
	}
	if view.BestScore != 18 {
		t.Fatalf("score = %d, want 18", view.BestScore)
	}
	if view.SubmissionID == nil {
		t.Fatal("missing best submission id")
	}
	if *view.SubmissionID != 2 {
		t.Fatalf("best submission id = %d, want 2", *view.SubmissionID)
	}
}

func TestMarkdownCodeFenceExpandsForEmbeddedFence(t *testing.T) {
	source := "fmt.Println(" + strings.Repeat(string(rune(96)), 3) + ")\n"
	block := markdownCodeBlock("go", source)
	wantPrefix := strings.Repeat(string(rune(96)), 4) + "go\n"
	if !strings.HasPrefix(block, wantPrefix) {
		t.Fatalf("markdown block did not expand fence: %q", block)
	}
	if !strings.Contains(block, source) {
		t.Fatalf("markdown block missing source: %q", block)
	}
}

func TestPreparedProblemInputDraftKeepsAssets(t *testing.T) {
	req := preparedProblemInput{
		Slug:          "image-prepared-problem",
		Title:         "Image prepared problem",
		Statement:     "![diagram](assets/diagram.png)",
		TimeLimitMS:   1000,
		MemoryLimitMB: 256,
		OutputLimitKB: 1024,
		Assets: []services.ProblemAssetDraft{
			{
				Name:        "diagram.png",
				Path:        "assets/diagram.png",
				ContentType: "image/png",
				Data:        "data:image/png;base64,iVBORw0KGgo=",
			},
		},
		Cases: []services.ProblemCaseDraft{
			{Name: "case-01", Input: "1 2\n", Output: "3\n", Weight: 100},
		},
	}

	draft := req.draft()
	if len(draft.Assets) != 1 {
		t.Fatalf("expected one asset, got %d", len(draft.Assets))
	}
	if draft.Assets[0].Path != "assets/diagram.png" {
		t.Fatalf("unexpected asset path: %s", draft.Assets[0].Path)
	}
	if draft.Assets[0].Data == "" {
		t.Fatal("expected asset data to be preserved")
	}
}

func TestFirstAvailableProblemInternalSlugReusesDeletedGap(t *testing.T) {
	got := firstAvailableProblemInternalSlug([]string{"P000001", "legacy-title", "P000003"})
	if got != "P000002" {
		t.Fatalf("slug = %q, want P000002", got)
	}
}

func TestParseProblemInternalSlugRejectsNonInternalValues(t *testing.T) {
	if got := parseProblemInternalSlug("p000042"); got != 42 {
		t.Fatalf("index = %d, want 42", got)
	}
	for _, value := range []string{"two-sum", "P42", "T000042", "P000000"} {
		if got := parseProblemInternalSlug(value); got != 0 {
			t.Fatalf("parseProblemInternalSlug(%q) = %d, want 0", value, got)
		}
	}
}

func TestCanCreateProblemsUsesIndependentAuthorFlag(t *testing.T) {
	student := models.User{Role: models.RoleStudent}
	if canCreateProblems(student) {
		t.Fatal("plain student must not be allowed to create problems")
	}
	student.CanAuthor = true
	if !canCreateProblems(student) {
		t.Fatal("approved student should create problems without changing role")
	}
	if student.Role != models.RoleStudent {
		t.Fatalf("author permission changed base role to %q", student.Role)
	}
	teacher := models.User{Role: models.RoleTeacher}
	if canCreateProblems(teacher) {
		t.Fatal("teacher without independent author permission must not create problems")
	}
	teacher.CanAuthor = true
	if !canCreateProblems(teacher) {
		t.Fatal("approved teacher should create problems")
	}
	if !canCreateProblems(models.User{Role: models.RoleAdmin}) {
		t.Fatal("administrator should always create problems")
	}
}

func TestPublicProblemSQLRequiresApprovedReview(t *testing.T) {
	sql := publicProblemSQL()
	for _, want := range []string{"problems.team_id IS NULL", "problem_reviews", "status <> 'approved'", "status = 'approved'"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("public problem SQL missing %q: %s", want, sql)
		}
	}
}

func TestProblemDifficultyVocabulary(t *testing.T) {
	for _, value := range []string{"入门", "基础", "普及", "提高", "综合", "挑战"} {
		if got := normalizeProblemDifficulty(value, nil); got != value {
			t.Fatalf("normalizeProblemDifficulty(%q) = %q", value, got)
		}
	}
	if got := normalizeProblemDifficulty("challenge", nil); got != "挑战" {
		t.Fatalf("challenge normalized to %q", got)
	}
	if got := normalizeProblemDifficulty("", datatypes.JSONMap{"labels": []string{"动态规划 DP", "挑战"}}); got != "挑战" {
		t.Fatalf("difficulty from tags = %q", got)
	}
	if got := normalizeProblemDifficulty("", nil); got != "入门" {
		t.Fatalf("empty difficulty = %q", got)
	}
}

func TestTeamSlugPattern(t *testing.T) {
	for _, value := range []string{"acm", "team-2026", "a1"} {
		if !teamSlugPattern.MatchString(value) {
			t.Fatalf("valid slug rejected: %s", value)
		}
	}
	for _, value := range []string{"Acm", "1team", "a", "team_name"} {
		if teamSlugPattern.MatchString(value) {
			t.Fatalf("invalid slug accepted: %s", value)
		}
	}
}

func TestProblemReviewViewsIncludeTestPointCount(t *testing.T) {
	views := problemReviewViews([]models.ProblemReview{{
		Problem: models.Problem{Manifest: datatypes.JSONMap{
			"cases": []any{
				map[string]any{"name": "case-01", "input": "tests/01.in", "output": "tests/01.out"},
				map[string]any{"name": "case-02", "input": "tests/02.in", "output": "tests/02.out"},
			},
		}},
	}})
	if len(views) != 1 || views[0].TestPointCount != 2 {
		t.Fatalf("test point count = %+v, want 2", views)
	}
}

func TestBuildXLSXIncludesExamReportRows(t *testing.T) {
	body, err := buildXLSX([][]xlsxCell{
		{xlsxString("学生姓名"), xlsxString("学号"), xlsxString("通过题目数"), xlsxString("所得分数")},
		{xlsxString("张三"), xlsxString("20260001"), xlsxNumber(2), xlsxNumber(180)},
	})
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	var sheet string
	for _, file := range zr.File {
		if file.Name != "xl/worksheets/sheet1.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		sheet = string(raw)
	}
	if !strings.Contains(sheet, "张三") || !strings.Contains(sheet, "<v>180</v>") {
		t.Fatalf("worksheet does not contain report row: %s", sheet)
	}
}

func TestRenderExamMarkdownReportIncludesSubmissionCode(t *testing.T) {
	now := time.Date(2026, 6, 13, 10, 30, 0, 0, time.UTC)
	item := models.Exam{
		ID:    9,
		Title: "期末考试",
		Problems: []models.ExamProblem{
			{ProblemID: 7, Label: "A", Score: 100, Problem: models.Problem{ID: 7, DisplayCode: "T001", Title: "A+B"}},
		},
	}
	students := []models.User{{ID: 2, Name: "张三", StudentNo: "20260001"}}
	rows := []examRankingRow{
		{Rank: 1, UserID: 2, Name: "张三", StudentNo: "20260001", TotalScore: 100, MaxScore: 100, Solved: 1, Attempted: 1, SubmissionCount: 1, ScoreReady: true, WorkStatus: "submitted", Problems: []examRankingCell{{ProblemID: 7, BestScore: 100, MaxScore: 100, Status: models.StatusAccepted, ScoreReady: true}}},
	}
	submissions := []models.Submission{
		{ID: 5, UserID: 2, ProblemID: 7, Language: "cpp", SourceCode: "int main() { return 0; }\n", Status: models.StatusAccepted, Score: 100, CreatedAt: now},
	}
	md := renderExamMarkdownReport(item, students, rows, submissions)
	for _, want := range []string{"# 考试归档：期末考试", "### 1. 张三（20260001）", "##### 提交 #5", "int main() { return 0; }"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
}

func TestNormalizeTeamProblemLabelsUsesStablePPrefix(t *testing.T) {
	links := []models.TeamProblemSetProblem{{ID: 11, SortOrder: 1}, {ID: 4, Label: "CUSTOM", SortOrder: 0}}
	normalizeTeamProblemLabels(links)
	if links[0].Label != "CUSTOM" {
		t.Fatalf("explicit label changed to %q", links[0].Label)
	}
	if links[1].Label != "P1011" {
		t.Fatalf("generated problem-set label = %q, want P1011", links[1].Label)
	}
}

func TestTeamContestWindow(t *testing.T) {
	start := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	contest := models.TeamContest{StartsAt: &start, DurationMinutes: 120, State: models.TeamContestPublished}
	if _, status := teamContestWindow(contest, start.Add(-time.Second)); status != models.TeamContestPublished {
		t.Fatalf("before start status = %q", status)
	}
	if endsAt, status := teamContestWindow(contest, start.Add(time.Hour)); status != models.TeamContestRunning || endsAt == nil || !endsAt.Equal(start.Add(2*time.Hour)) {
		t.Fatalf("running window = %v, %q", endsAt, status)
	}
	if _, status := teamContestWindow(contest, start.Add(2*time.Hour)); status != models.TeamContestClosed {
		t.Fatalf("at end status = %q", status)
	}
	contest.State = models.TeamContestDraft
	if _, status := teamContestWindow(contest, start.Add(time.Hour)); status != models.TeamContestDraft {
		t.Fatalf("draft contest advanced to %q", status)
	}
}

func TestBuildTeamContestRankingUsesAggregates(t *testing.T) {
	start := time.Date(2026, 8, 6, 9, 0, 0, 0, time.UTC)
	solvedAt := start.Add(35 * time.Minute)
	users := []models.User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
	links := []models.TeamContestProblem{{ProblemID: 11}, {ProblemID: 12}}
	aggregates := []teamContestCellAggregate{
		{UserID: 1, ProblemID: 11, Attempts: 3, WrongAttempts: 2, BestScore: 100, SolvedAt: &solvedAt, LastSubmission: solvedAt, LatestStatus: string(models.StatusAccepted)},
		{UserID: 2, ProblemID: 11, Attempts: 1, BestScore: 40, LastSubmission: start.Add(10 * time.Minute), LatestStatus: string(models.StatusWrongAnswer)},
	}
	rows := buildTeamContestRanking(users, links, aggregates, start, "penalty")
	if len(rows) != 2 || rows[0].UserID != 1 {
		t.Fatalf("ranking order = %#v", rows)
	}
	if rows[0].Solved != 1 || rows[0].PenaltyMinutes != 75 || rows[0].SubmissionCount != 3 {
		t.Fatalf("aggregate totals = %#v", rows[0])
	}
	if !rows[0].Problems[0].Fastest || rows[1].Problems[0].Status != string(models.StatusWrongAnswer) {
		t.Fatalf("problem cells = %#v / %#v", rows[0].Problems[0], rows[1].Problems[0])
	}
}

func TestElapsedContestMinutesUsesWholeMinutesAndClampsEarlySubmissions(t *testing.T) {
	start := time.Date(2026, 8, 6, 9, 0, 0, 0, time.UTC)
	if got := elapsedContestMinutes(start, start.Add(17*time.Minute+59*time.Second)); got != 17 {
		t.Fatalf("elapsed minutes = %d, want 17", got)
	}
	if got := elapsedContestMinutes(start, start.Add(-time.Second)); got != 0 {
		t.Fatalf("early submission minutes = %d, want 0", got)
	}
}
