package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"school-oj/apps/api/internal/models"
)

type catalogQueryLog struct {
	logger.Interface
	queries *[]string
}

func (l catalogQueryLog) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	*l.queries = append(*l.queries, sql)
}

func TestCatalogBatchesStatisticsAndPreservesRates(t *testing.T) {
	f := newExamFixture(t)
	for i := 0; i < 20; i++ {
		problem := models.Problem{OwnerID: f.teacher.ID, Slug: fmt.Sprintf("catalog-%d", i), DisplayCode: fmt.Sprintf("C%03d", i), Title: "Catalog test", Difficulty: "入门"}
		if err := f.s.DB.Create(&problem).Error; err != nil {
			t.Fatal(err)
		}
		for _, status := range []models.SubmissionStatus{models.StatusAccepted, models.StatusWrongAnswer, models.StatusQueued, models.StatusRunning, models.StatusPendingReview, models.StatusSystemError} {
			sub := models.Submission{UserID: f.student.ID, ProblemID: problem.ID, ProblemVersionID: f.version.ID, Language: "python", SourceCode: "print(3)", Status: status}
			if err := f.s.DB.Create(&sub).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	queries := []string{}
	s := f.s
	s.DB = s.DB.Session(&gorm.Session{Logger: catalogQueryLog{Interface: logger.Default, queries: &queries}})
	router := gin.New()
	router.GET("/catalog", s.listProblemCatalog)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/catalog?page_size=20", nil))
	if response.Code != 200 {
		t.Fatalf("catalog: %d %s", response.Code, response.Body.String())
	}
	var body struct {
		Items []problemCatalogItem `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 20 {
		t.Fatalf("items=%d", len(body.Items))
	}
	for _, item := range body.Items {
		if item.AcceptedCount != 1 || item.EvaluatedCount != 2 || item.PassRate != 50 {
			t.Fatalf("wrong counts for %d: accepted=%d evaluated=%d rate=%f", item.ID, item.AcceptedCount, item.EvaluatedCount, item.PassRate)
		}
	}
	if len(queries) != 4 {
		t.Fatalf("expected 4 queries independent of page size, got %d: %v", len(queries), queries)
	}
	if !strings.Contains(queries[2], "GROUP BY") {
		t.Fatalf("missing grouped statistics: %v", queries)
	}
}

func TestCatalogRejectsUnboundedFilters(t *testing.T) {
	router := gin.New()
	router.GET("/catalog", (Server{DB: &gorm.DB{}}).listProblemCatalog)
	// This must be rejected before the first database query/session.
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/catalog?page=2147483647", nil))
	if response.Code != 400 {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}
