package handlers

import (
	"gorm.io/gorm"
	"school-oj/apps/api/internal/models"
	"time"
)

func (p eventPaper) lifecycle(now time.Time) string {
	if p.kind == "exam" && p.exam.DeletedAt != nil || p.kind == "contest" && p.contest.DeletedAt != nil {
		return "deleted"
	}
	return p.review().Lifecycle(p.starts(), p.ends(), now)
}

type eventCompletionChecks struct {
	Judging  int64 `json:"judging"`
	Ungraded int64 `json:"ungraded"`
	Failed   int64 `json:"failed"`
}

func completionChecks(db *gorm.DB, p eventPaper) (eventCompletionChecks, error) {
	var checks eventCompletionChecks
	column := "exam_id"
	if p.kind == "contest" {
		column = "team_contest_id"
	}
	if err := db.Model(&models.Submission{}).Where(column+"=? AND status IN ?", p.id, []models.SubmissionStatus{models.StatusQueued, models.StatusRunning}).Count(&checks.Judging).Error; err != nil {
		return checks, err
	}
	if err := db.Model(&models.Submission{}).Where(column+"=? AND status=?", p.id, models.StatusSystemError).Count(&checks.Failed).Error; err != nil {
		return checks, err
	}
	if p.kind == "exam" && examManualReview(p.exam) {
		if err := db.Model(&models.Submission{}).Where("exam_id=? AND manual_graded_at IS NULL", p.id).Count(&checks.Ungraded).Error; err != nil {
			return checks, err
		}
	}
	return checks, nil
}

// Call while holding the event row lock, so a grading/rejudge transaction
// cannot begin between the final checks and publication.
func validateEventCompletion(db *gorm.DB, p eventPaper, now time.Time) error {
	if p.lifecycle(now) != "closed" || p.kind == "contest" && p.contest.State == models.TeamContestDraft {
		return eventFail(409, "仅已通过赛前审核且已截止的场次可以确认完结")
	}
	checks, err := completionChecks(db, p)
	if err != nil {
		return err
	}
	if checks.Judging > 0 {
		return eventFail(409, "仍有提交正在排队或评测，请等待判题完成")
	}
	if checks.Ungraded > 0 {
		return eventFail(409, "仍有人工评阅未完成，请完成评分后再确认完结")
	}
	if checks.Failed > 0 {
		return eventFail(409, "仍有判题系统异常，请处理并重新评测后再确认完结")
	}
	return nil
}
