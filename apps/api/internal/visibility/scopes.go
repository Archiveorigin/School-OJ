// Package visibility is L3: SQL predicates run before count, pagination and
// projection. Its inputs are the authenticated principal, never client roles.
package visibility

import (
	"gorm.io/gorm"
	"school-oj/apps/api/internal/authz"
	"school-oj/apps/api/internal/models"
)

func TeachingCourseIDs(db *gorm.DB, user models.User) *gorm.DB {
	q := db.Model(&models.Course{}).Select("courses.id")
	if authz.Has(user, authz.CourseAny) {
		return q
	}
	if !authz.Has(user, authz.CourseTeach) {
		return q.Where("FALSE")
	}
	return q.Where("courses.teacher_id = ? OR EXISTS (SELECT 1 FROM course_memberships cm WHERE cm.course_id = courses.id AND cm.user_id = ? AND cm.role IN ?)", user.ID, user.ID, []string{"teacher", "admin", "course_admin", "course_assistant"})
}

func Submissions(user models.User) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if authz.Has(user, authz.SubmissionAny) {
			return db
		}
		if user.ID == 0 || user.AccountDeleted || !authz.ValidBaseRole(user.Role) {
			return db.Where("FALSE")
		}
		courses := TeachingCourseIDs(db.Session(&gorm.Session{NewDB: true}), user)
		return db.Where(`user_id = ?
 OR (assignment_id IS NULL AND exam_id IS NULL AND team_contest_id IS NULL AND problem_set_id IS NULL
     AND problem_id IN (SELECT id FROM problems WHERE owner_id = ? AND deleted_at IS NULL))
 OR (team_contest_id IS NULL AND problem_set_id IS NULL AND assignment_id IN (SELECT id FROM assignments WHERE deleted_at IS NULL AND course_id IN (?)))
 OR (team_contest_id IS NULL AND problem_set_id IS NULL AND exam_id IN (SELECT id FROM exams WHERE deleted_at IS NULL AND course_id IN (?)))
 OR team_contest_id IN (SELECT tc.id FROM team_contests tc JOIN teams t ON t.id=tc.team_id JOIN team_memberships tm ON tm.team_id=t.id AND tm.user_id=?
     WHERE tc.deleted_at IS NULL AND (tm.role IN ('owner','admin') OR tc.created_by=? AND tm.role='member' AND t.contest_permission='all'))
 OR problem_set_id IN (SELECT ps.id FROM team_problem_sets ps JOIN teams t ON t.id=ps.team_id JOIN team_memberships tm ON tm.team_id=t.id AND tm.user_id=?
     WHERE ps.deleted_at IS NULL AND (tm.role IN ('owner','admin') OR ps.created_by=? AND tm.role='member' AND t.contest_permission='all'))`,
			user.ID, user.ID, courses, courses, user.ID, user.ID, user.ID, user.ID)
	}
}

// Public statistics must not reveal attempts made in unpublished exams,
// private teams or assignments even when they reuse a public problem.
func PublicPractice(db *gorm.DB) *gorm.DB {
	return db.Where("assignment_id IS NULL AND exam_id IS NULL AND team_contest_id IS NULL AND problem_set_id IS NULL")
}

func PreparedProblems(user models.User) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if authz.Has(user, authz.ProblemAny) {
			return db
		}
		if !authz.Has(user, authz.CourseTeach) {
			return db.Where("FALSE")
		}
		return db.Where("prepared_problems.owner_id = ?", user.ID)
	}
}

func Courses(user models.User) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if authz.Has(user, authz.CourseAny) {
			return db
		}
		clean := db.Session(&gorm.Session{NewDB: true})
		if authz.Has(user, authz.CourseTeach) {
			return db.Where("id IN (?)", TeachingCourseIDs(clean, user))
		}
		if user.ID == 0 || !authz.ValidBaseRole(user.Role) {
			return db.Where("FALSE")
		}
		return db.Omit("join_code").Where("id IN (?)", clean.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role = ?", user.ID, models.RoleStudent))
	}
}

func Classes(user models.User) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if authz.Has(user, authz.CourseAny) {
			return db
		}
		clean := db.Session(&gorm.Session{NewDB: true})
		if authz.Has(user, authz.CourseTeach) {
			return db.Where("course_id IN (?)", TeachingCourseIDs(clean, user))
		}
		if user.ID == 0 || !authz.ValidBaseRole(user.Role) {
			return db.Where("FALSE")
		}
		return db.Omit("join_code").Where("id IN (?)", clean.Model(&models.ClassMembership{}).Select("class_id").Where("user_id = ?", user.ID))
	}
}

func CourseWork(user models.User) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if authz.Has(user, authz.CourseAny) {
			return db
		}
		clean := db.Session(&gorm.Session{NewDB: true})
		if authz.Has(user, authz.CourseTeach) {
			return db.Where("course_id IN (?)", TeachingCourseIDs(clean, user))
		}
		if user.ID == 0 || !authz.ValidBaseRole(user.Role) {
			return db.Where("FALSE")
		}
		return db.Where("class_id IN (?) OR (class_id IS NULL AND course_id IN (?))",
			clean.Model(&models.ClassMembership{}).Select("class_id").Where("user_id = ?", user.ID),
			clean.Model(&models.CourseMembership{}).Select("course_id").Where("user_id = ? AND role = ?", user.ID, models.RoleStudent))
	}
}
