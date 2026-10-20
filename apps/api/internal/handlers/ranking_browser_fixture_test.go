package handlers

import (
	"fmt"
	"testing"
	"time"

	"school-oj/apps/api/internal/models"
)

type rankingBrowserCase struct {
	Kind   string `json:"kind"`
	ID     uint   `json:"id"`
	Rule   string `json:"rule"`
	Queued uint   `json:"queued"`
}

// Only used by the opt-in loopback browser fixture, never by the API binary.
func seedRankingBrowserCases(t *testing.T, f *examTestFixture) []rankingBrowserCase {
	t.Helper()
	start, end := time.Now().Add(-2*time.Hour), time.Now().Add(-time.Hour)
	team := models.Team{Name: "排名验收团队", Slug: "ranking-browser-qa", OwnerID: f.teacher.ID, ContestPermission: "all"}
	if err := f.s.DB.Create(&team).Error; err != nil {
		t.Fatal(err)
	}
	for _, user := range []models.User{f.teacher, f.student, f.peer} {
		role := models.TeamRoleMember
		if user.ID == f.teacher.ID {
			role = models.TeamRoleOwner
		}
		f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: user.ID, Role: role})
	}
	problems := []models.Problem{f.problem}
	versions := []models.ProblemVersion{f.version}
	for i, title := range []string{"字符串排序", "网格路径", "区间统计"} {
		problem := models.Problem{OwnerID: f.teacher.ID, Slug: fmt.Sprintf("ranking-qa-%d", i), Title: title, DisplayCode: fmt.Sprintf("QA%d", i), Manifest: f.problem.Manifest}
		if err := f.s.DB.Create(&problem).Error; err != nil {
			t.Fatal(err)
		}
		version := models.ProblemVersion{ProblemID: problem.ID, Version: 1, Title: title, Manifest: f.version.Manifest, CreatedBy: f.teacher.ID}
		if err := f.s.DB.Create(&version).Error; err != nil {
			t.Fatal(err)
		}
		f.s.DB.Model(&problem).Update("current_version_id", version.ID)
		problems, versions = append(problems, problem), append(versions, version)
	}
	result := []rankingBrowserCase{}
	for _, kind := range []string{"exam", "contest"} {
		for _, rule := range []string{"acm", "oi", "ioi"} {
			p := eventPaper{kind: kind}
			title := "春季算法挑战赛"
			if kind == "exam" {
				title = "程序设计阶段测试"
			}
			review := models.EventReview{ReviewStatus: "approved", ReviewRevision: 1}
			if kind == "exam" {
				p.exam = models.Exam{CourseID: f.exam.CourseID, Title: title, StartsAt: &start, EndsAt: &end, ScoringRule: rule, EventReview: review, ParticipantsFrozen: true}
				if err := f.s.DB.Create(&p.exam).Error; err != nil {
					t.Fatal(err)
				}
				p.id = p.exam.ID
				for _, user := range []models.User{f.student, f.peer} {
					f.s.DB.Create(&models.ExamParticipant{ExamID: p.id, UserID: user.ID})
				}
				for i, problem := range problems {
					f.s.DB.Create(&models.ExamProblem{ExamID: p.id, ProblemID: problem.ID, ProblemVersionID: versions[i].ID, Label: defaultProblemLabel(i), Score: 100, SortOrder: i})
				}
			} else {
				p.contest = models.TeamContest{TeamID: team.ID, Title: title, StartsAt: &start, DurationMinutes: 60, ScoringRule: rule, State: models.TeamContestPublished, CreatedBy: f.teacher.ID, EventReview: review}
				if err := f.s.DB.Create(&p.contest).Error; err != nil {
					t.Fatal(err)
				}
				p.id = p.contest.ID
				for _, user := range []models.User{f.student, f.peer, f.teacher} {
					f.s.DB.Create(&models.TeamContestParticipant{ContestID: p.id, UserID: user.ID})
				}
				for i, problem := range problems {
					f.s.DB.Create(&models.TeamContestProblem{ContestID: p.id, ProblemID: problem.ID, ProblemVersionID: versions[i].ID, Label: defaultProblemLabel(i), SortOrder: i})
				}
			}
			rankingSubmission(t, f, p, f.student, models.StatusCompileError, 0, start.Add(time.Minute))
			rankingSubmission(t, f, p, f.student, models.StatusWrongAnswer, 40, start.Add(2*time.Minute))
			rankingSubmission(t, f, p, f.student, models.StatusAccepted, 100, start.Add(10*time.Minute))
			rankingSubmission(t, f, p, f.student, models.StatusWrongAnswer, 20, start.Add(50*time.Minute))
			rankingSubmission(t, f, p, f.peer, models.StatusWrongAnswer, 50, start.Add(30*time.Minute))
			queued := rankingSubmission(t, f, p, f.peer, models.StatusQueued, 0, start.Add(55*time.Minute))
			second := rankingSubmission(t, f, p, f.peer, models.StatusWrongAnswer, 50, start.Add(20*time.Minute))
			f.s.DB.Model(&second).Updates(map[string]any{"problem_id": problems[1].ID, "problem_version_id": versions[1].ID})
			result = append(result, rankingBrowserCase{kind, p.id, rule, queued.ID})
		}
	}
	return result
}
