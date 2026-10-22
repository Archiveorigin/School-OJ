package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"school-oj/apps/api/internal/models"
	"testing"
)

func seedContestUIBrowserCases(t *testing.T, f *examTestFixture, contestID uint) gin.H {
	t.Helper()
	p, err := loadEventPaper(f.s.DB, "contest", contestID, false)
	if err != nil {
		t.Fatal(err)
	}
	set := models.TeamProblemSet{TeamID: p.contest.TeamID, Title: "团队训练题单", CreatedBy: f.teacher.ID}
	if err := f.s.DB.Create(&set).Error; err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 27; i++ {
		problem := models.Problem{OwnerID: f.teacher.ID, Slug: fmt.Sprintf("picker-qa-%d", i), DisplayCode: fmt.Sprintf("PICK%02d", i), Title: fmt.Sprintf("选题验收 %02d", i), Statement: f.problem.Statement, Manifest: f.problem.Manifest, PackageObject: f.problem.PackageObject}
		if i > 24 {
			problem.TeamID = &p.contest.TeamID
		}
		if err := f.s.DB.Create(&problem).Error; err != nil {
			t.Fatal(err)
		}
		version := models.ProblemVersion{ProblemID: problem.ID, Version: 1, Title: problem.Title, Statement: problem.Statement, Manifest: f.version.Manifest, CreatedBy: f.teacher.ID, PackageObject: f.version.PackageObject}
		if err := f.s.DB.Create(&version).Error; err != nil {
			t.Fatal(err)
		}
		if err := f.s.DB.Model(&problem).Update("current_version_id", version.ID).Error; err != nil {
			t.Fatal(err)
		}
		if i > 24 {
			if err := f.s.DB.Create(&models.TeamProblemSetProblem{ProblemSetID: set.ID, ProblemID: problem.ID, Label: fmt.Sprint(i)}).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	return gin.H{"team_id": p.contest.TeamID, "problem_set_id": set.ID}
}
