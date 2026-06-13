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
