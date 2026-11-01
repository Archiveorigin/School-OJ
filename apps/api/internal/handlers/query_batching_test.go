package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"school-oj/apps/api/internal/models"
)

// countQueries returns how many SELECT statements fn issues on the fixture database.
func countQueries(t *testing.T, f *examTestFixture, fn func()) int64 {
	t.Helper()
	var count atomic.Int64
	name := fmt.Sprintf("test:count:%d", time.Now().UnixNano())
	increment := func(*gorm.DB) { count.Add(1) }
	if err := f.s.DB.Callback().Query().Before("gorm:query").Register(name, increment); err != nil {
		t.Fatal(err)
	}
	if err := f.s.DB.Callback().Row().Before("gorm:row").Register(name, increment); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = f.s.DB.Callback().Query().Remove(name)
		_ = f.s.DB.Callback().Row().Remove(name)
	}()
	fn()
	return count.Load()
}

// listsSubmission reports whether a JSON list response contains the submission.
func listsSubmission(t *testing.T, body []byte, id uint) bool {
	t.Helper()
	var rows []struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatal(err, string(body))
	}
	for _, row := range rows {
		if row.ID == id {
			return true
		}
	}
	return false
}

func addStudents(t *testing.T, f *examTestFixture, courseID uint, count int) []models.User {
	t.Helper()
	users := make([]models.User, count)
	for i := range users {
		users[i] = models.User{Email: fmt.Sprintf("batch-%d-%d@example.test", i, time.Now().UnixNano()), Name: fmt.Sprintf("批量考生%d", i), Role: models.RoleStudent, PasswordHash: "unused", EmailVerified: true}
		if err := f.s.DB.Create(&users[i]).Error; err != nil {
			t.Fatal(err)
		}
		if err := f.s.DB.Create(&models.CourseMembership{CourseID: courseID, UserID: users[i].ID, Role: models.RoleStudent}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return users
}

func TestRoutineAdministratorReadsAreNotAuditedPostgres(t *testing.T) {
	f := newExamFixture(t)
	allowRows := func(route string) int64 {
		var count int64
		if err := f.s.DB.Model(&models.AuditLog{}).Where("action = 'authorization.allow' AND meta->>'route' = ?", route).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		return count
	}
	assertStatus(t, f.request("GET", "/api/teams", &f.admin, nil), 200)
	if rows := allowRows("/api/teams"); rows != 0 {
		t.Fatalf("routine administrator read audited %d times", rows)
	}
	// Capability-gated reads, writes and submission access stay on record.
	assertStatus(t, f.request("GET", "/api/users", &f.admin, nil), 200)
	if allowRows("/api/users") != 1 {
		t.Fatal("privileged read was not audited")
	}
	sub := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Language: "python", SourceCode: "print(1)", Status: models.StatusAccepted}
	if err := f.s.DB.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	assertStatus(t, f.request("GET", fmt.Sprintf("/api/submissions/%d", sub.ID), &f.admin, nil), 200)
	if allowRows("/api/submissions/:id") != 1 {
		t.Fatal("submission access was not audited")
	}
	assertStatus(t, f.request("PUT", "/api/profile", &f.admin, gin.H{"name": "管理员改名"}), 200)
	if allowRows("/api/profile") != 1 {
		t.Fatal("write was not audited")
	}
	assertStatus(t, f.request("GET", "/api/users", &f.student, nil), 403)
	var denied int64
	f.s.DB.Model(&models.AuditLog{}).Where("action = 'authorization.deny'").Count(&denied)
	if denied != 1 {
		t.Fatal("denied request was not audited", denied)
	}
}

func TestAvatarLoadedOnlyWhereReturnedPostgres(t *testing.T) {
	f := newExamFixture(t)
	avatar := "data:image/png;base64," + strings.Repeat("A", 4096)
	if err := f.s.DB.Model(&f.student).Update("avatar_url", avatar).Error; err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/me", "/api/profile"} {
		response := f.request("GET", path, &f.student, nil)
		assertStatus(t, response, 200)
		if !strings.Contains(response.Body.String(), avatar) {
			t.Fatal(path, "lost the avatar")
		}
	}
	// A profile edit runs with the avatar-less session user and must not erase it.
	assertStatus(t, f.request("PUT", "/api/profile", &f.student, gin.H{"name": "新名字"}), 200)
	var stored models.User
	if err := f.s.DB.First(&stored, f.student.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.AvatarURL != avatar || stored.Name != "新名字" {
		t.Fatal("profile update changed the avatar", stored.Name, len(stored.AvatarURL))
	}
	// Standings never need avatars.
	students, err := f.s.loadExamRankingStudents(f.exam)
	if err != nil || len(students) == 0 {
		t.Fatal(err, len(students))
	}
	for _, student := range students {
		if student.AvatarURL != "" {
			t.Fatal("ranking loaded an avatar")
		}
	}
}

func TestSubmissionListsCarryLengthNotSourcePostgres(t *testing.T) {
	f := newExamFixture(t)
	source := "# 两数之和\nprint(3)\n"
	sub := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Language: "python", SourceCode: source, Status: models.StatusAccepted, Score: 100}
	if err := f.s.DB.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.s.DB.Create(&models.SubmissionResult{SubmissionID: sub.ID, CaseName: "one", Status: models.StatusAccepted, Message: "ok"}).Error; err != nil {
		t.Fatal(err)
	}
	failed := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Language: "python", SourceCode: "x", Status: models.StatusWrongAnswer}
	if err := f.s.DB.Create(&failed).Error; err != nil {
		t.Fatal(err)
	}
	for _, result := range []models.SubmissionResult{{SubmissionID: failed.ID, CaseName: "one", Status: models.StatusAccepted}, {SubmissionID: failed.ID, CaseName: "two", Status: models.StatusWrongAnswer}} {
		if err := f.s.DB.Create(&result).Error; err != nil {
			t.Fatal(err)
		}
	}
	response := f.request("GET", "/api/submissions?mine=true", &f.student, nil)
	assertStatus(t, response, 200)
	var rows []submissionListView
	if err := json.Unmarshal(response.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatal("rows", len(rows))
	}
	for _, row := range rows {
		if row.SourceCode != "" {
			t.Fatal("list leaked source", row.ID)
		}
		switch row.ID {
		case sub.ID:
			if row.CodeLength != len([]rune(source)) || row.ErrorPoint != "" {
				t.Fatal("wrong list row", row.CodeLength, row.ErrorPoint)
			}
		case failed.ID:
			if row.CodeLength != 1 || !strings.HasSuffix(row.ErrorPoint, "2") {
				t.Fatal("wrong failure row", row.CodeLength, row.ErrorPoint)
			}
		}
	}
	// The editor restores from /latest and the detail view shows code: both keep it.
	latest := f.request("GET", fmt.Sprintf("/api/submissions/latest?standalone=true&problem_id=%d", f.problem.ID), &f.student, nil)
	assertStatus(t, latest, 200)
	if !strings.Contains(latest.Body.String(), "两数之和") && !strings.Contains(latest.Body.String(), `"source_code":"x"`) {
		t.Fatal("latest lost source", latest.Body.String())
	}
	detail := f.request("GET", fmt.Sprintf("/api/submissions/%d", sub.ID), &f.student, nil)
	assertStatus(t, detail, 200)
	if !strings.Contains(detail.Body.String(), "两数之和") {
		t.Fatal("detail lost source")
	}
	profile := f.request("GET", "/api/profile", &f.student, nil)
	assertStatus(t, profile, 200)
	if strings.Contains(profile.Body.String(), "两数之和") {
		t.Fatal("profile recent list leaked source")
	}
}

// Results must equal what the former per-student loop produced: newest-first ties,
// best-score selection, per-rule scoring, and pending handling.
func TestExamReportSummariesPerStudentPostgres(t *testing.T) {
	f := newExamFixture(t)
	start := time.Now().Add(-30 * time.Minute)
	add := func(user models.User, status models.SubmissionStatus, score int, offset time.Duration) models.Submission {
		sub := models.Submission{UserID: user.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, ExamID: &f.exam.ID, Language: "python", SourceCode: "secret-source", Status: status, Score: score, CreatedAt: start.Add(offset)}
		if err := f.s.DB.Create(&sub).Error; err != nil {
			t.Fatal(err)
		}
		return sub
	}
	// student: best attempt is the older one. peer: only a pending attempt.
	best := add(f.student, models.StatusWrongAnswer, 60, time.Minute)
	add(f.student, models.StatusWrongAnswer, 20, 2*time.Minute)
	add(f.peer, models.StatusQueued, 0, time.Minute)
	if err := f.s.DB.Create(&models.ExamAttempt{ExamID: f.exam.ID, UserID: f.student.ID}).Error; err != nil {
		t.Fatal(err)
	}

	response := f.request("GET", fmt.Sprintf("/api/exams/%d/report", f.exam.ID), &f.teacher, nil)
	assertStatus(t, response, 200)
	if strings.Contains(response.Body.String(), "secret-source") {
		t.Fatal("report carried source code")
	}
	var report struct {
		Rows []struct {
			User          models.User        `json:"user"`
			WorkStatus    string             `json:"work_status"`
			TotalScore    int                `json:"total_score"`
			MaxScore      int                `json:"max_score"`
			ScoreReady    bool               `json:"score_ready"`
			ProblemScores []problemScoreView `json:"problem_scores"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	seen := map[uint]bool{}
	for _, row := range report.Rows {
		seen[row.User.ID] = true
		if row.MaxScore != 35 || len(row.ProblemScores) != 1 || row.ProblemScores[0].Label != "B" {
			t.Fatalf("report row shape: %+v", row)
		}
		switch row.User.ID {
		case f.student.ID:
			// (60*35+50)/100 = 21, taken from the older, better attempt.
			if row.WorkStatus != "submitted" || row.TotalScore != 21 || !row.ScoreReady || *row.ProblemScores[0].SubmissionID != best.ID {
				t.Fatalf("student row: %+v", row)
			}
		case f.peer.ID:
			if row.WorkStatus != "submitted" || row.TotalScore != 0 || row.ScoreReady || !row.ProblemScores[0].PendingReview {
				t.Fatalf("peer row: %+v", row)
			}
		}
	}
	if !seen[f.student.ID] || !seen[f.peer.ID] {
		t.Fatal("students missing from report")
	}
	// A student with nothing recorded still appears as unattempted.
	if summary := f.s.examSummary(f.other.ID, f.student.ID, false); summary.WorkStatus != "unattempted" || summary.MaxScore != 35 {
		t.Fatalf("empty exam summary: %+v", summary)
	}
	if summary := f.s.examSummary(f.exam.ID, f.student.ID, false); summary.WorkStatus != "submitted" {
		t.Fatalf("single summary: %+v", summary)
	}
}

func TestAssignmentSummariesPostgres(t *testing.T) {
	f := newExamFixture(t)
	var course models.Course
	if err := f.s.DB.First(&course).Error; err != nil {
		t.Fatal(err)
	}
	assignment := models.Assignment{CourseID: course.ID, Title: "批量作业"}
	if err := f.s.DB.Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.s.DB.Create(&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: f.problem.ID, Score: 50}).Error; err != nil {
		t.Fatal(err)
	}
	sub := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, AssignmentID: &assignment.ID, Language: "python", SourceCode: "x", Status: models.StatusAccepted, Score: 100}
	if err := f.s.DB.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	// Practice on the same problem is not part of the assignment.
	practice := models.Submission{UserID: f.peer.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Language: "python", SourceCode: "x", Status: models.StatusAccepted, Score: 100}
	if err := f.s.DB.Create(&practice).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.s.DB.Create(&models.AssignmentAttempt{AssignmentID: assignment.ID, UserID: f.peer.ID}).Error; err != nil {
		t.Fatal(err)
	}
	summaries := f.s.assignmentSummaries([]uint{assignment.ID}, []uint{f.student.ID, f.peer.ID, f.teacher.ID}, true)[assignment.ID]
	if got := summaries[f.student.ID]; got.WorkStatus != "submitted" || got.TotalScore != 50 || got.MaxScore != 50 || !got.ScoreReady || len(got.Problems) != 1 {
		t.Fatalf("submitted student: %+v", got)
	}
	if got := summaries[f.peer.ID]; got.WorkStatus != "unsubmitted" || got.TotalScore != 0 {
		t.Fatalf("attempted without submitting: %+v", got)
	}
	if got := summaries[f.teacher.ID]; got.WorkStatus != "unattempted" {
		t.Fatalf("untouched: %+v", got)
	}
	if empty := f.s.assignmentSummaries([]uint{assignment.ID}, nil, false); len(empty[assignment.ID]) != 0 {
		t.Fatal("no users must yield no summaries")
	}
	if got := f.s.assignmentSummary(assignment.ID, f.student.ID, false); got.TotalScore != 50 || got.Problems != nil {
		t.Fatalf("single summary: %+v", got)
	}
}

func TestReportQueriesDoNotScaleWithStudentsPostgres(t *testing.T) {
	f := newExamFixture(t)
	var course models.Course
	if err := f.s.DB.First(&course).Error; err != nil {
		t.Fatal(err)
	}
	report := fmt.Sprintf("/api/exams/%d/report", f.exam.ID)
	before := countQueries(t, f, func() { assertStatus(t, f.request("GET", report, &f.teacher, nil), 200) })
	for _, student := range addStudents(t, f, course.ID, 12) {
		sub := models.Submission{UserID: student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, ExamID: &f.exam.ID, Language: "python", SourceCode: "x", Status: models.StatusAccepted, Score: 100}
		if err := f.s.DB.Create(&sub).Error; err != nil {
			t.Fatal(err)
		}
	}
	after := countQueries(t, f, func() { assertStatus(t, f.request("GET", report, &f.teacher, nil), 200) })
	if after != before {
		t.Fatalf("exam report queries grew with students: %d -> %d", before, after)
	}
}

func TestContestPollingQueriesDoNotScalePostgres(t *testing.T) {
	f, _, p, start := rankingFixture(t, "contest", "acm")
	if err := f.s.DB.Model(p.model()).Update("starts_at", time.Now().Add(-10*time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	list := fmt.Sprintf("/api/contests/%d/submissions", p.id)
	detail := fmt.Sprintf("/api/contests/%d", p.id)
	rankingSubmission(t, f, p, f.student, models.StatusWrongAnswer, 0, start)
	listBefore := countQueries(t, f, func() { assertStatus(t, f.request("GET", list, &f.student, nil), 200) })
	detailBefore := countQueries(t, f, func() { assertStatus(t, f.request("GET", detail, &f.student, nil), 200) })
	for i := 0; i < 15; i++ {
		rankingSubmission(t, f, p, f.student, models.StatusWrongAnswer, 0, start.Add(time.Duration(i)*time.Second))
	}
	listAfter := countQueries(t, f, func() { assertStatus(t, f.request("GET", list, &f.student, nil), 200) })
	if listAfter != listBefore {
		t.Fatalf("contest list queries grew with rows: %d -> %d", listBefore, listAfter)
	}
	// Adding problems must not add per-problem submission lookups.
	second := models.Problem{OwnerID: f.teacher.ID, Slug: "batch-second", Title: "第二题", DisplayCode: "T002", Difficulty: "入门"}
	if err := f.s.DB.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	version := models.ProblemVersion{ProblemID: second.ID, Version: 1, Title: second.Title, CreatedBy: f.teacher.ID}
	if err := f.s.DB.Create(&version).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.s.DB.Create(&models.TeamContestProblem{ContestID: p.id, ProblemID: second.ID, ProblemVersionID: version.ID, Label: "B"}).Error; err != nil {
		t.Fatal(err)
	}
	detailAfter := countQueries(t, f, func() { assertStatus(t, f.request("GET", detail, &f.student, nil), 200) })
	if detailAfter != detailBefore {
		t.Fatalf("contest detail queries grew with problems: %d -> %d", detailBefore, detailAfter)
	}
}

func TestContestDetailSubmissionStatesPostgres(t *testing.T) {
	type detailBody struct {
		Problems []teamContestProblemView `json:"problems"`
	}
	read := func(t *testing.T, f *examTestFixture, id uint) teamContestProblemView {
		t.Helper()
		response := f.request("GET", fmt.Sprintf("/api/contests/%d", id), &f.student, nil)
		assertStatus(t, response, 200)
		var body detailBody
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Problems) != 1 {
			t.Fatal("problems", len(body.Problems))
		}
		return body.Problems[0]
	}
	t.Run("acm: an accepted attempt outranks a later failure", func(t *testing.T) {
		f, _, p, _ := rankingFixture(t, "contest", "acm")
		now := time.Now()
		if err := f.s.DB.Model(p.model()).Update("starts_at", now.Add(-10*time.Minute)).Error; err != nil {
			t.Fatal(err)
		}
		if got := read(t, f, p.id); got.SubmissionStatus != "" || got.SubmittedAt != nil {
			t.Fatalf("no submission yet: %+v", got)
		}
		rankingSubmission(t, f, p, f.student, models.StatusAccepted, 100, now.Add(-5*time.Minute))
		rankingSubmission(t, f, p, f.student, models.StatusWrongAnswer, 0, now.Add(-time.Minute))
		rankingSubmission(t, f, p, f.peer, models.StatusWrongAnswer, 0, now)
		got := read(t, f, p.id)
		if got.SubmissionStatus != string(models.StatusAccepted) || got.SubmittedAt == nil || got.SubmittedAt.Before(now.Add(-2*time.Minute)) {
			t.Fatalf("own latest state: %+v", got)
		}
	})
	t.Run("oi: results stay embargoed for the author", func(t *testing.T) {
		f, _, p, _ := rankingFixture(t, "contest", "oi")
		now := time.Now()
		if err := f.s.DB.Model(p.model()).Update("starts_at", now.Add(-10*time.Minute)).Error; err != nil {
			t.Fatal(err)
		}
		// With the fixture's "all" permission every member organizes and may see results.
		if err := f.s.DB.Model(&models.Team{}).Where("id = ?", p.contest.TeamID).Update("contest_permission", "owner").Error; err != nil {
			t.Fatal(err)
		}
		rankingSubmission(t, f, p, f.student, models.StatusAccepted, 100, now.Add(-time.Minute))
		if got := read(t, f, p.id); got.SubmissionStatus != "recorded" {
			t.Fatalf("embargo broken: %+v", got)
		}
	})
}

func TestProblemSetDetailCountsPostgres(t *testing.T) {
	f := newExamFixture(t)
	team := models.Team{Name: "批量题单", Slug: "batch-set", OwnerID: f.teacher.ID, ContestPermission: "all"}
	if err := f.s.DB.Create(&team).Error; err != nil {
		t.Fatal(err)
	}
	for _, member := range []models.User{f.teacher, f.student, f.peer} {
		role := models.TeamRoleMember
		if member.ID == f.teacher.ID {
			role = models.TeamRoleOwner
		}
		if err := f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: member.ID, Role: role}).Error; err != nil {
			t.Fatal(err)
		}
	}
	set := models.TeamProblemSet{TeamID: team.ID, Title: "题单", CreatedBy: f.teacher.ID}
	if err := f.s.DB.Create(&set).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.s.DB.Create(&models.TeamProblemSetProblem{ProblemSetID: set.ID, ProblemID: f.problem.ID, Label: "A"}).Error; err != nil {
		t.Fatal(err)
	}
	add := func(user models.User, status models.SubmissionStatus) {
		sub := models.Submission{UserID: user.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, ProblemSetID: &set.ID, Language: "python", SourceCode: "private-team-source", Status: status}
		if err := f.s.DB.Create(&sub).Error; err != nil {
			t.Fatal(err)
		}
	}
	add(f.student, models.StatusAccepted)
	add(f.student, models.StatusWrongAnswer)
	add(f.peer, models.StatusWrongAnswer)
	response := f.request("GET", fmt.Sprintf("/api/teams/%d/problem-sets/%d", team.ID, set.ID), &f.student, nil)
	assertStatus(t, response, 200)
	var body struct {
		Problems []teamProblemLinkView `json:"problems"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Problems) != 1 {
		t.Fatal("problems", response.Body.String())
	}
	got := body.Problems[0]
	if got.SubmissionCount != 3 || got.AcceptedCount != 1 || got.SubmissionStatus != string(models.StatusAccepted) || got.SubmittedAt == nil {
		t.Fatalf("set row: %+v", got)
	}
	list := f.request("GET", fmt.Sprintf("/api/teams/%d/problem-sets/%d/submissions", team.ID, set.ID), &f.student, nil)
	assertStatus(t, list, 200)
	if strings.Contains(list.Body.String(), "private-team-source") {
		t.Fatal("set list carried source")
	}
}

func TestAnnouncementAttachmentsGroupedPostgres(t *testing.T) {
	f := newExamFixture(t)
	base, id := eventTestPaper(t, f, "exam")
	p, err := loadEventPaper(f.s.DB, "exam", id, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.s.DB.Model(p.model()).Updates(map[string]any{"review_status": "approved", "starts_at": time.Now().Add(-time.Minute)}).Error; err != nil {
		t.Fatal(err)
	}
	var withFile, plain uint
	for index, target := range []*uint{&withFile, &plain} {
		item := eventAnnouncement{EventKind: "exam", EventID: id, Body: fmt.Sprintf("公告%d", index), CreatedBy: f.teacher.ID}
		if err := f.s.DB.Create(&item).Error; err != nil {
			t.Fatal(err)
		}
		*target = item.ID
	}
	attachment := models.EventAttachment{AnnouncementID: &withFile, EventKind: "exam", EventID: id, Name: "a.txt", Object: "k", Size: 1, Checksum: "c", CreatedBy: f.teacher.ID}
	if err := f.s.DB.Create(&attachment).Error; err != nil {
		t.Fatal(err)
	}
	response := f.request("GET", base+"/announcements", &f.teacher, nil)
	assertStatus(t, response, 200)
	var items []struct {
		ID          uint              `json:"id"`
		Attachments []json.RawMessage `json:"attachments"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.Body.String(), `"attachments":[]`) {
		t.Fatal("an announcement without files must serialize an empty array", response.Body.String())
	}
	for _, item := range items {
		want := 0
		if item.ID == withFile {
			want = 1
		}
		if len(item.Attachments) != want {
			t.Fatal("attachment grouped under wrong announcement", item.ID, len(item.Attachments))
		}
	}
}
