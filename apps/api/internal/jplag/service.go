package jplag

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"school-oj/apps/api/internal/config"
	"school-oj/apps/api/internal/models"

	"github.com/minio/minio-go/v7"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service struct {
	DB    *gorm.DB
	MinIO *minio.Client
	Cfg   config.Config
}

func (s Service) Run(ctx context.Context, jobID uint) {
	var job models.PlagiarismJob
	if err := s.DB.First(&job, jobID).Error; err != nil {
		return
	}
	s.DB.Model(&job).Updates(map[string]any{"status": "running", "message": "collecting submissions"})

	var submissions []models.Submission
	query := s.DB.Where("language = ? AND status = ?", job.Language, models.StatusAccepted)
	if job.AssignmentID != nil {
		query = query.Where("assignment_id = ?", *job.AssignmentID)
	}
	if job.ExamID != nil {
		query = query.Where("exam_id = ?", *job.ExamID)
	}
	if err := query.Find(&submissions).Error; err != nil {
		s.fail(&job, err)
		return
	}
	if len(submissions) < 2 {
		s.DB.Model(&job).Updates(map[string]any{
			"status":  "completed",
			"message": "not enough accepted submissions for comparison",
			"summary": datatypes.JSONMap{"submissions": len(submissions), "matches": []any{}},
		})
		return
	}

	if err := os.MkdirAll(s.Cfg.JPlagWorkDir, 0o755); err != nil {
		s.fail(&job, err)
		return
	}
	workDir, err := os.MkdirTemp(s.Cfg.JPlagWorkDir, fmt.Sprintf("job-%d-", job.ID))
	if err != nil {
		s.fail(&job, err)
		return
	}
	defer os.RemoveAll(workDir)
	sourceDir := filepath.Join(workDir, "sources")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		s.fail(&job, err)
		return
	}
	ext := extension(job.Language)
	for _, sub := range submissions {
		name := fmt.Sprintf("u%d_s%d%s", sub.UserID, sub.ID, ext)
		if err := os.WriteFile(filepath.Join(sourceDir, name), []byte(sub.SourceCode), 0o644); err != nil {
			s.fail(&job, err)
			return
		}
	}

	if s.Cfg.JPlagJarPath != "" {
		reportPath := filepath.Join(workDir, "jplag-report.zip")
		jplagLang := language(job.Language)
		cmd := exec.CommandContext(ctx, "java", "-jar", s.Cfg.JPlagJarPath, "-l", jplagLang, "-r", reportPath, sourceDir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			s.fail(&job, fmt.Errorf("jplag failed: %w: %s", err, trimOutput(out)))
			return
		}
		body, err := os.ReadFile(reportPath)
		if err != nil {
			s.fail(&job, err)
			return
		}
		object := fmt.Sprintf("plagiarism/job-%d/report.zip", job.ID)
		if _, err := s.MinIO.PutObject(ctx, s.Cfg.MinIOBucket, object, bytes.NewReader(body), int64(len(body)), minio.PutObjectOptions{ContentType: "application/zip"}); err != nil {
			s.fail(&job, err)
			return
		}
		s.DB.Model(&job).Updates(map[string]any{
			"status":        "completed",
			"report_object": object,
			"message":       "jplag report generated",
			"summary":       datatypes.JSONMap{"submissions": len(submissions), "mode": "jplag"},
		})
		return
	}

	report, matches := lightweightReport(submissions)
	object := fmt.Sprintf("plagiarism/job-%d/report.zip", job.ID)
	if _, err := s.MinIO.PutObject(ctx, s.Cfg.MinIOBucket, object, bytes.NewReader(report), int64(len(report)), minio.PutObjectOptions{ContentType: "application/zip"}); err != nil {
		s.fail(&job, err)
		return
	}
	s.DB.Model(&job).Updates(map[string]any{
		"status":        "completed",
		"report_object": object,
		"message":       "JPLAG_JAR_PATH is not configured; generated a token-overlap report",
		"summary":       datatypes.JSONMap{"submissions": len(submissions), "matches": matches, "mode": "fallback"},
	})
}

func (s Service) fail(job *models.PlagiarismJob, err error) {
	s.DB.Model(job).Updates(map[string]any{"status": "failed", "message": err.Error()})
}

func extension(language string) string {
	switch language {
	case "c":
		return ".c"
	case "cpp":
		return ".cpp"
	case "java":
		return ".java"
	default:
		return ".py"
	}
}

func language(language string) string {
	switch language {
	case "cpp":
		return "cpp"
	case "c":
		return "c"
	case "java":
		return "java"
	default:
		return "python3"
	}
}

func trimOutput(out []byte) string {
	s := strings.TrimSpace(string(out))
	if len(s) > 2000 {
		return s[:2000]
	}
	return s
}

func lightweightReport(submissions []models.Submission) ([]byte, []map[string]any) {
	type pair struct {
		A     uint    `json:"a"`
		B     uint    `json:"b"`
		Score float64 `json:"score"`
	}
	var pairs []pair
	for i := 0; i < len(submissions); i++ {
		for j := i + 1; j < len(submissions); j++ {
			score := overlap(submissions[i].SourceCode, submissions[j].SourceCode)
			if score >= 0.65 {
				pairs = append(pairs, pair{A: submissions[i].ID, B: submissions[j].ID, Score: score})
			}
		}
	}
	payload := map[string]any{
		"generated_at": time.Now().Format(time.RFC3339),
		"engine":       "fallback-token-overlap",
		"matches":      pairs,
	}
	raw, _ := json.MarshalIndent(payload, "", "  ")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("report.json")
	_, _ = w.Write(raw)
	_ = zw.Close()
	out := make([]map[string]any, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, map[string]any{"a": p.A, "b": p.B, "score": p.Score})
	}
	return buf.Bytes(), out
}

func overlap(a, b string) float64 {
	as := tokens(a)
	bs := tokens(b)
	if len(as) == 0 || len(bs) == 0 {
		return 0
	}
	var common int
	for tok := range as {
		if bs[tok] {
			common++
		}
	}
	minLen := len(as)
	if len(bs) < minLen {
		minLen = len(bs)
	}
	return float64(common) / float64(minLen)
}

func tokens(s string) map[string]bool {
	out := map[string]bool{}
	for _, tok := range strings.FieldsFunc(s, func(r rune) bool {
		return !(r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	}) {
		if len(tok) > 1 {
			out[tok] = true
		}
	}
	return out
}
