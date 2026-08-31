package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRouterBuilds(t *testing.T) {
	_ = (Server{}).Router()
}

func TestSortExamRankingRowsByScore(t *testing.T) {
	base := time.Date(2026, 6, 7, 10, 0, 0, 0, time.UTC)
	later := base.Add(time.Minute)
	finished := base.Add(2 * time.Minute)
	rows := []examRankingRow{
		{Name: "Charlie", studentNo: "S3", TotalScore: 80, Solved: 2, LastSubmission: &later},
		{Name: "Alice", studentNo: "S1", TotalScore: 100, Solved: 1, LastSubmission: &later},
		{Name: "Bob", studentNo: "S2", TotalScore: 100, Solved: 2, LastSubmission: &later, FinishedAt: &finished},
		{Name: "Ada", studentNo: "S0", TotalScore: 100, Solved: 2, LastSubmission: &base},
	}

	sortExamRankingRows(rows, "score")

	got := []string{rows[0].Name, rows[1].Name, rows[2].Name, rows[3].Name}
	want := []string{"Ada", "Bob", "Alice", "Charlie"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rank %d = %s, want %s; full order=%v", i+1, got[i], want[i], got)
		}
	}
}

func TestSortExamRankingRowsByPenalty(t *testing.T) {
	rows := []examRankingRow{
		{Name: "One", Solved: 1, PenaltyMinutes: 10, TotalScore: 100},
		{Name: "Slow", Solved: 2, PenaltyMinutes: 90, TotalScore: 300},
		{Name: "Fast", Solved: 2, PenaltyMinutes: 45, TotalScore: 10},
	}

	sortExamRankingRows(rows, "penalty")

	got := []string{rows[0].Name, rows[1].Name, rows[2].Name}
	want := []string{"Fast", "Slow", "One"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("rank %d = %s, want %s; full order=%v", index+1, got[index], want[index], got)
		}
	}
}

func TestExamScoringRuleDefaultsAndRejectsInvalidInput(t *testing.T) {
	for _, test := range []struct {
		name string
		exam models.Exam
		want string
	}{
		{name: "missing", exam: models.Exam{}, want: "acm"},
		{name: "legacy score", exam: models.Exam{ScoringRule: "score"}, want: "ioi"},
		{name: "legacy penalty", exam: models.Exam{ScoringRule: "penalty"}, want: "acm"},
		{name: "oi", exam: models.Exam{ScoringRule: "oi"}, want: "oi"},
		{name: "ioi", exam: models.Exam{ScoringRule: "ioi"}, want: "ioi"},
		{name: "acm", exam: models.Exam{ScoringRule: "acm"}, want: "acm"},
		{name: "invalid", exam: models.Exam{ScoringRule: "points"}, want: "acm"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := examScoringRule(test.exam); got != test.want {
				t.Fatalf("examScoringRule() = %q, want %q", got, test.want)
			}
		})
	}
	if got, valid := parseScoringRule(""); !valid || got != "acm" {
		t.Fatalf("empty scoring rule = %q, %v; want acm, true", got, valid)
	}
	if got, valid := parseScoringRule("not-a-rule"); valid || got != "" {
		t.Fatalf("invalid scoring rule = %q, %v; want empty, false", got, valid)
	}
}

func TestCreateExamRejectsInvalidScoringRule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/exams", strings.NewReader(`{"title":"Invalid rule","scoring_rule":"points"}`))
	context.Request.Header.Set("Content-Type", "application/json")

	(Server{}).createExam(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestExamPenaltyStatsUsesFirstAcceptedAndIgnoresPendingFailures(t *testing.T) {
	start := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	submissions := []models.Submission{
		{ID: 8, Status: models.StatusSystemError, CreatedAt: start.Add(2 * time.Minute)},
		{ID: 7, Status: models.StatusWrongAnswer, CreatedAt: start.Add(50 * time.Minute)},
		{ID: 6, Status: models.StatusAccepted, CreatedAt: start.Add(35 * time.Minute)},
		{ID: 5, Status: models.StatusCompileError, CreatedAt: start.Add(20 * time.Minute)},
		{ID: 4, Status: models.StatusRunning, CreatedAt: start.Add(15 * time.Minute)},
		{ID: 3, Status: models.StatusPendingReview, CreatedAt: start.Add(12 * time.Minute)},
		{ID: 2, Status: models.StatusQueued, CreatedAt: start.Add(10 * time.Minute)},
		{ID: 1, Status: models.StatusWrongAnswer, CreatedAt: start.Add(5 * time.Minute)},
	}

	stats := examPenaltyStats(submissions, start, false, 100)
	if stats.Attempts != 6 || stats.WrongAttempts != 2 || stats.ElapsedMinutes != 35 {
		t.Fatalf("penalty stats = %#v, want attempts=6 wrong=2 elapsed=35", stats)
	}
	if stats.FirstAccepted == nil || !stats.FirstAccepted.Equal(start.Add(35*time.Minute)) {
		t.Fatalf("first accepted = %v", stats.FirstAccepted)
	}
	if got := stats.ElapsedMinutes + stats.WrongAttempts*20; got != 75 {
		t.Fatalf("penalty = %d, want 75", got)
	}
}

func TestExamPenaltyStatsUsesFirstFullManualGradeAsAccepted(t *testing.T) {
	start := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	partial := 60
	full := 100
	pending := models.Submission{ID: 1, Status: models.StatusPendingReview, CreatedAt: start.Add(5 * time.Minute)}
	submissions := []models.Submission{
		pending,
		{ID: 2, Status: models.StatusManualGraded, ManualScore: &partial, CreatedAt: start.Add(12 * time.Minute)},
		{ID: 3, Status: models.StatusManualGraded, ManualScore: &full, CreatedAt: start.Add(30 * time.Minute)},
		{ID: 4, Status: models.StatusPendingReview, CreatedAt: start.Add(40 * time.Minute)},
	}

	stats := examPenaltyStats(submissions, start, true, 100)
	if stats.Attempts != 3 || stats.WrongAttempts != 1 || stats.ElapsedMinutes != 30 {
		t.Fatalf("manual penalty stats = %#v, want attempts=3 wrong=1 elapsed=30", stats)
	}
	if stats.FirstAccepted == nil || !stats.FirstAccepted.Equal(start.Add(30*time.Minute)) {
		t.Fatalf("first manually accepted = %v", stats.FirstAccepted)
	}
	if isExamPenaltyAccepted(pending, true, 100) {
		t.Fatal("an ungraded manual-review submission must not count as accepted")
	}
}

func TestExamRankingAccessReasonBlocksViewerBeforeStart(t *testing.T) {
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	start := now.Add(time.Hour)
	exam := models.Exam{StartsAt: &start}

	if got := examRankingAccessReason(exam, false, true, true, now); got != "exam has not started" {
		t.Fatalf("viewer access reason = %q, want exam has not started", got)
	}
	if got := examRankingAccessReason(exam, true, false, false, now); got != "" {
		t.Fatalf("manager access reason = %q, want empty", got)
	}
	if got := examRankingAccessReason(exam, false, true, true, start); got != "" {
		t.Fatalf("started exam access reason = %q, want empty", got)
	}
	if got := examRankingAccessReason(exam, false, false, true, start); got != "forbidden" {
		t.Fatalf("hidden ranking access reason = %q, want forbidden", got)
	}
}

func TestExamRankingCellPendingDoesNotOverrideFullScore(t *testing.T) {
	if !isExamFullScore(100, 100) || isExamFullScore(60, 100) {
		t.Fatal("full-score detection must be independent of later pending submissions")
	}
	if examRankingCellPending("score", true, 100, 100) {
		t.Fatal("a later pending submission must not hide an already-full score")
	}
	if !examRankingCellPending("score", true, 60, 100) {
		t.Fatal("a pending submission must remain visible while it can improve a partial score")
	}
	if !examRankingCellPending("penalty", true, 100, 100) {
		t.Fatal("penalty pending state is resolved from first acceptance separately")
	}
}

func TestOIProblemScoreUsesLastSubmission(t *testing.T) {
	start := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	older := models.Submission{ID: 1, Status: models.StatusAccepted, Score: 100, CreatedAt: start}
	latest := models.Submission{ID: 2, Status: models.StatusWrongAnswer, Score: 20, CreatedAt: start.Add(time.Minute)}
	view, submitted, pending := problemScoreFromSubmissionsForRule(models.Problem{ID: 7}, 100, false, []models.Submission{older, latest}, "oi")
	if !submitted || pending || view.SubmissionID == nil || *view.SubmissionID != latest.ID || view.BestScore != 20 {
		t.Fatalf("OI last-submission score = %#v, submitted=%v pending=%v", view, submitted, pending)
	}
}

func TestSubmissionVisibilityForOIAndFreeze(t *testing.T) {
	now := time.Date(2026, 8, 31, 10, 30, 0, 0, time.UTC)
	start := now.Add(-90 * time.Minute)
	end := now.Add(30 * time.Minute)
	if !examSubmissionHidden(models.Exam{ScoringRule: "oi", EndsAt: &end}, now.Add(-time.Hour), now) {
		t.Fatal("OI result must be hidden before the exam ends")
	}
	if examSubmissionHidden(models.Exam{ScoringRule: "oi", EndsAt: &end}, now, end) {
		t.Fatal("OI result must be public after the exam ends")
	}
	contest := models.TeamContest{ScoringRule: "acm", StartsAt: &start, DurationMinutes: 120, State: models.TeamContestRunning, FreezeEnabled: true, FreezeDurationMinutes: 60}
	freezeAt := end.Add(-time.Hour)
	if teamContestSubmissionHidden(contest, freezeAt.Add(-time.Second), now) {
		t.Fatal("pre-freeze result must remain visible in the frozen snapshot")
	}
	if !teamContestSubmissionHidden(contest, freezeAt, now) {
		t.Fatal("submission at the freeze boundary must be hidden")
	}
}

func TestApplyProblemVersionUsesImmutableSnapshot(t *testing.T) {
	problem := models.Problem{Title: "current", PackageObject: "current.zip", TimeLimitMS: 1000}
	version := models.ProblemVersion{ID: 4, Title: "event snapshot", PackageObject: "v1.zip", TimeLimitMS: 2500}
	applyProblemVersion(&problem, version)
	if problem.Title != "event snapshot" || problem.PackageObject != "v1.zip" || problem.TimeLimitMS != 2500 {
		t.Fatalf("hydrated problem = %#v", problem)
	}
}

func TestStandaloneProblemSubmissionExcludesScopedWork(t *testing.T) {
	if !isStandaloneProblemSubmission(models.Submission{IsPublic: true}) {
		t.Fatal("public practice submission should be standalone")
	}
	contestID := uint(9)
	if isStandaloneProblemSubmission(models.Submission{IsPublic: true, TeamContestID: &contestID}) {
		t.Fatal("team contest submission must not be exposed as standalone public practice")
	}
	problemSetID := uint(10)
	if isStandaloneProblemSubmission(models.Submission{IsPublic: true, ProblemSetID: &problemSetID}) {
		t.Fatal("problem-set submission must not be exposed as standalone public practice")
	}
}

func TestMarkExamFastestCells(t *testing.T) {
	start := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	fast := start.Add(10 * time.Minute)
	slow := start.Add(20 * time.Minute)
	rows := []examRankingRow{
		{Problems: []examRankingCell{{ProblemID: 1, firstAccepted: &slow}}},
		{Problems: []examRankingCell{{ProblemID: 1, firstAccepted: &fast}}},
	}

	markExamFastestCells(rows)

	if rows[0].Problems[0].Fastest || !rows[1].Problems[0].Fastest {
		t.Fatalf("fastest flags = %v, %v", rows[0].Problems[0].Fastest, rows[1].Problems[0].Fastest)
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
	for _, want := range []string{"problems.archived_at IS NULL", "problems.team_id IS NULL", "problem_reviews", "status <> 'approved'", "status = 'approved'"} {
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
		{Rank: 1, UserID: 2, Name: "张三", TotalScore: 100, MaxScore: 100, Solved: 1, Attempted: 1, SubmissionCount: 1, ScoreReady: true, WorkStatus: "submitted", Problems: []examRankingCell{{ProblemID: 7, BestScore: 100, MaxScore: 100, Status: models.StatusAccepted, ScoreReady: true}}, studentNo: "20260001"},
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

func TestExamRankingRowJSONKeepsNameAndHidesStudentNumber(t *testing.T) {
	payload, err := json.Marshal(examRankingRow{
		Rank:      1,
		UserID:    2,
		Name:      "张三",
		studentNo: "20260001",
	})
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(payload)
	if !strings.Contains(serialized, `"name":"张三"`) {
		t.Fatalf("exam ranking JSON must retain the participant name: %s", serialized)
	}
	if strings.Contains(serialized, "student_no") || strings.Contains(serialized, "20260001") {
		t.Fatalf("exam ranking JSON must not expose student numbers: %s", serialized)
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

func TestTeamContestPublishErrorResponseDistinguishesDeletedAndEmpty(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantMessage string
		wantHandled bool
	}{
		{name: "deleted", err: fmt.Errorf("wrapped: %w", errTeamContestDeleted), wantStatus: http.StatusConflict, wantMessage: "比赛已删除，请刷新后重试", wantHandled: true},
		{name: "empty", err: errTeamContestNoProblem, wantStatus: http.StatusBadRequest, wantMessage: "发布前至少添加一道题目", wantHandled: true},
		{name: "invalid start", err: gorm.ErrInvalidData, wantStatus: http.StatusBadRequest, wantMessage: "发布前必须设置晚于当前时间的开始时间", wantHandled: true},
		{name: "unexpected", err: errors.New("database unavailable")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, message, handled := teamContestPublishErrorResponse(test.err)
			if status != test.wantStatus || message != test.wantMessage || handled != test.wantHandled {
				t.Fatalf("teamContestPublishErrorResponse() = (%d, %q, %v), want (%d, %q, %v)", status, message, handled, test.wantStatus, test.wantMessage, test.wantHandled)
			}
		})
	}
}

func TestActiveTeamProblemSetLockQueryRequiresActiveRowAndShareLock(t *testing.T) {
	database, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=localhost user=test dbname=test sslmode=disable",
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	sql := database.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var set models.TeamProblemSet
		return activeTeamProblemSetLockQuery(tx, 42).First(&set)
	})
	for _, required := range []string{"deleted_at IS NULL", "FOR SHARE"} {
		if !strings.Contains(sql, required) {
			t.Fatalf("active problem-set lock query missing %q: %s", required, sql)
		}
	}
}

func TestResolveTeamContestAwardPercentages(t *testing.T) {
	zero := 0
	twenty := 20
	gold, silver, bronze, err := resolveTeamContestAwardPercentages(nil, nil, nil, 10, 10, 10)
	if err != nil || gold != 10 || silver != 10 || bronze != 10 {
		t.Fatalf("default award percentages = %d/%d/%d, err=%v", gold, silver, bronze, err)
	}
	gold, silver, bronze, err = resolveTeamContestAwardPercentages(&zero, &twenty, nil, 10, 10, 10)
	if err != nil || gold != 0 || silver != 20 || bronze != 10 {
		t.Fatalf("explicit/partial award percentages = %d/%d/%d, err=%v", gold, silver, bronze, err)
	}

	overflow := 81
	if _, _, _, err := resolveTeamContestAwardPercentages(&overflow, &twenty, &zero, 10, 10, 10); err == nil {
		t.Fatal("award percentages totaling over 100 must be rejected")
	}
	outOfRange := -1
	if _, _, _, err := resolveTeamContestAwardPercentages(&outOfRange, nil, nil, 10, 10, 10); err == nil {
		t.Fatal("negative award percentage must be rejected")
	}
}

func TestTeamContestAwardPercentagesSerializeAsNumbers(t *testing.T) {
	zero := 0
	ten := 10
	payload, err := json.Marshal(models.TeamContest{
		GoldAwardPercent:   &zero,
		SilverAwardPercent: &ten,
		BronzeAwardPercent: &ten,
	})
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(payload)
	for _, want := range []string{`"gold_award_percent":0`, `"silver_award_percent":10`, `"bronze_award_percent":10`} {
		if !strings.Contains(serialized, want) {
			t.Fatalf("serialized contest missing %s: %s", want, serialized)
		}
	}
}

func TestTeamManagementPermissionRejectsOrdinaryMembers(t *testing.T) {
	tests := []struct {
		name           string
		userRole       models.Role
		membershipRole models.TeamRole
		joined         bool
		want           bool
	}{
		{name: "system admin", userRole: models.RoleAdmin, want: true},
		{name: "team owner", userRole: models.RoleStudent, membershipRole: models.TeamRoleOwner, joined: true, want: true},
		{name: "team admin", userRole: models.RoleStudent, membershipRole: models.TeamRoleAdmin, joined: true, want: true},
		{name: "ordinary member", userRole: models.RoleStudent, membershipRole: models.TeamRoleMember, joined: true, want: false},
		{name: "not joined", userRole: models.RoleStudent, membershipRole: models.TeamRoleAdmin, joined: false, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hasTeamManagerPermission(test.userRole, test.membershipRole, test.joined); got != test.want {
				t.Fatalf("hasTeamManagerPermission() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestTeamManagementRoutesAreRegistered(t *testing.T) {
	routes := (Server{}).Router().Routes()
	want := map[string]bool{
		http.MethodPut + " /api/teams/:id/contests/:contest_id":    false,
		http.MethodDelete + " /api/teams/:id/contests/:contest_id": false,
		http.MethodPut + " /api/contests/:contest_id":              false,
		http.MethodDelete + " /api/contests/:contest_id":           false,
		http.MethodPut + " /api/teams/:id/problem-sets/:set_id":    false,
		http.MethodDelete + " /api/teams/:id/problem-sets/:set_id": false,
		http.MethodPut + " /api/problem-sets/:set_id":              false,
		http.MethodDelete + " /api/problem-sets/:set_id":           false,
	}
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("management route %s is not registered", route)
		}
	}
}

func TestBuildTeamContestRankingUsesAggregates(t *testing.T) {
	start := time.Date(2026, 8, 6, 9, 0, 0, 0, time.UTC)
	solvedAt := start.Add(35 * time.Minute)
	users := []models.User{{ID: 1, Name: "Alice", StudentNo: "S001"}, {ID: 2, Name: "Bob", StudentNo: "S002"}}
	links := []models.TeamContestProblem{{ProblemID: 11}, {ProblemID: 12}}
	aggregates := []teamContestCellAggregate{
		{UserID: 1, ProblemID: 11, Attempts: 3, SubmissionCount: 4, WrongAttempts: 2, BestScore: 100, SolvedAt: &solvedAt, LastSubmission: solvedAt, LatestStatus: string(models.StatusAccepted)},
		{UserID: 2, ProblemID: 11, Attempts: 1, SubmissionCount: 1, BestScore: 40, LastSubmission: start.Add(10 * time.Minute), LatestStatus: string(models.StatusWrongAnswer)},
	}
	rows := buildTeamContestRanking(users, links, aggregates, start, "penalty")
	if len(rows) != 2 || rows[0].UserID != 1 {
		t.Fatalf("ranking order = %#v", rows)
	}
	if rows[0].Solved != 1 || rows[0].PenaltyMinutes != 75 || rows[0].SubmissionCount != 4 {
		t.Fatalf("aggregate totals = %#v", rows[0])
	}
	payload, err := json.Marshal(rows[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "student_no") || strings.Contains(string(payload), "S001") {
		t.Fatalf("team ranking must not expose student numbers: %s", payload)
	}
	if rows[0].MaxScore != 200 || rows[0].Problems[0].MaxScore != 100 {
		t.Fatalf("score maxima = row %d, cell %d", rows[0].MaxScore, rows[0].Problems[0].MaxScore)
	}
	if !rows[0].Problems[0].Fastest || rows[1].Problems[0].Status != string(models.StatusWrongAnswer) {
		t.Fatalf("problem cells = %#v / %#v", rows[0].Problems[0], rows[1].Problems[0])
	}
}

func TestActiveTeamContestUsersExcludesDeletedAccounts(t *testing.T) {
	users := []models.User{
		{ID: 1, Name: "Active"},
		{ID: 2, Name: "Deleted", AccountDeleted: true},
	}
	active := activeTeamContestUsers(users)
	if len(active) != 1 || active[0].ID != 1 {
		t.Fatalf("active team contest users = %#v, want only user 1", active)
	}
}

func TestBuildTeamContestRankingIncludesEligibleUsersWithoutSubmissions(t *testing.T) {
	start := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)
	users := []models.User{
		{ID: 1, Name: "Submitted"},
		{ID: 2, Name: "Zero submissions"},
	}
	links := []models.TeamContestProblem{{ProblemID: 11}}
	aggregates := []teamContestCellAggregate{{
		UserID:         1,
		ProblemID:      11,
		Attempts:       1,
		BestScore:      20,
		LastSubmission: start.Add(time.Minute),
		LatestStatus:   string(models.StatusWrongAnswer),
	}}

	rows := buildTeamContestRanking(users, links, aggregates, start, "penalty")
	if len(rows) != len(users) {
		t.Fatalf("ranking row count = %d, want all %d eligible users", len(rows), len(users))
	}
	var found bool
	for _, row := range rows {
		if row.UserID != 2 {
			continue
		}
		found = true
		if row.SubmissionCount != 0 || row.Solved != 0 || len(row.Problems) != 1 {
			t.Fatalf("zero-submission participant row = %#v", row)
		}
	}
	if !found {
		t.Fatal("eligible user without submissions is missing from ranking")
	}
}

func TestBuildTeamContestRankingScoreUsesTotalBeforeSolved(t *testing.T) {
	start := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	acceptedAt := start.Add(10 * time.Minute)
	users := []models.User{{ID: 1, Name: "More solved"}, {ID: 2, Name: "Higher score"}}
	links := []models.TeamContestProblem{{ProblemID: 11}, {ProblemID: 12}}
	aggregates := []teamContestCellAggregate{
		{UserID: 1, ProblemID: 11, Attempts: 1, BestScore: 30, SolvedAt: &acceptedAt, LastSubmission: acceptedAt, LatestStatus: string(models.StatusAccepted)},
		{UserID: 1, ProblemID: 12, Attempts: 1, BestScore: 30, SolvedAt: &acceptedAt, LastSubmission: acceptedAt, LatestStatus: string(models.StatusAccepted)},
		{UserID: 2, ProblemID: 11, Attempts: 1, BestScore: 100, SolvedAt: &acceptedAt, LastSubmission: acceptedAt, LatestStatus: string(models.StatusAccepted)},
	}

	rows := buildTeamContestRanking(users, links, aggregates, start, "score")
	if len(rows) != 2 || rows[0].UserID != 2 {
		t.Fatalf("score ranking = %#v; total score must precede solved count", rows)
	}
}

func TestBuildTeamContestRankingIOIUsesLastEffectiveScoreTime(t *testing.T) {
	start := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	users := []models.User{{ID: 1, Name: "Later effective"}, {ID: 2, Name: "Earlier effective"}}
	links := []models.TeamContestProblem{{ProblemID: 11}}
	aggregates := []teamContestCellAggregate{
		{UserID: 1, ProblemID: 11, BestScore: 80, LastSubmission: start.Add(20 * time.Minute), EffectiveAt: start.Add(18 * time.Minute)},
		{UserID: 2, ProblemID: 11, BestScore: 80, LastSubmission: start.Add(30 * time.Minute), EffectiveAt: start.Add(12 * time.Minute)},
	}

	rows := buildTeamContestRanking(users, links, aggregates, start, "ioi")
	if len(rows) != 2 || rows[0].UserID != 2 {
		t.Fatalf("IOI ranking = %#v; earlier effective scoring time must win the tie", rows)
	}
}

func TestProblemScoreUsesEarliestSubmissionThatReachedBestScore(t *testing.T) {
	start := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	view, submitted, pending := problemScoreFromSubmissions(models.Problem{ID: 11}, 100, false, []models.Submission{
		{ID: 2, Score: 80, Status: models.StatusWrongAnswer, CreatedAt: start.Add(20 * time.Minute)},
		{ID: 1, Score: 80, Status: models.StatusWrongAnswer, CreatedAt: start.Add(10 * time.Minute)},
	})
	if !submitted || pending || view.SubmissionID == nil || *view.SubmissionID != 1 {
		t.Fatalf("best score view = %#v, submitted=%v pending=%v", view, submitted, pending)
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
