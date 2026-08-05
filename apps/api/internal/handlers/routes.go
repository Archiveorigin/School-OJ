package handlers

import (
	"fmt"
	"strings"
	"time"

	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (s Server) Router() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), middleware.SecurityHeaders())
	if err := router.SetTrustedProxies(s.Cfg.TrustedProxies); err != nil {
		panic(fmt.Errorf("configure trusted proxies: %w", err))
	}

	allowedOrigins := s.Cfg.CORSOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:3000", "http://localhost:5173"}
	}
	allowAllOrigins := len(allowedOrigins) == 1 && strings.TrimSpace(allowedOrigins[0]) == "*"
	if allowAllOrigins {
		allowedOrigins = nil
	}
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  allowAllOrigins,
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length", "X-RateLimit-Limit", "X-RateLimit-Remaining", "Retry-After"},
		AllowCredentials: !allowAllOrigins,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/healthz", s.health)
	s.registerAPIRoutes(router.Group("/api"))
	s.registerAPIRoutes(router.Group("/api/v1"))
	return router
}

func (s Server) registerAPIRoutes(api *gin.RouterGroup) {
	api.POST("/auth/login", middleware.RateLimitIP(s.Redis, "auth:login", 10, 15*time.Minute), s.login)
	api.POST("/auth/send-code", middleware.RateLimitIP(s.Redis, "auth:send-code", 5, 10*time.Minute), s.sendEmailCode)
	api.POST("/auth/register", middleware.RateLimitIP(s.Redis, "auth:register", 5, time.Hour), s.register)
	api.POST("/auth/password-reset", middleware.RateLimitIP(s.Redis, "auth:password-reset", 5, 15*time.Minute), s.resetPassword)
	api.GET("/problems", middleware.OptionalAuth(s.DB, s.Cfg.JWTSecret), s.listProblems)
	api.GET("/problems/:id/assets/*asset_path", middleware.OptionalAuth(s.DB, s.Cfg.JWTSecret), s.getProblemAsset)
	api.GET("/problems/:id", middleware.OptionalAuth(s.DB, s.Cfg.JWTSecret), s.getProblem)

	auth := api.Group("")
	auth.Use(middleware.Auth(s.DB, s.Cfg.JWTSecret))
	auth.GET("/me", s.me)
	auth.GET("/me/active-exam", s.activeExam)
	auth.GET("/profile", s.getProfile)
	auth.PUT("/profile", s.updateProfile)
	auth.POST("/profile/password", s.updateProfilePassword)
	auth.POST("/profile/email-code", middleware.RateLimitUser(s.Redis, "profile:email-code", 5, 10*time.Minute), s.sendProfileEmailCode)
	auth.POST("/profile/email", s.rebindEmail)
	auth.DELETE("/profile", s.deleteProfile)
	auth.GET("/author-applications/me", s.getMyAuthorApplication)
	auth.POST("/author-applications", s.createAuthorApplication)
	auth.POST("/feedback", s.createFeedback)
	auth.GET("/teams", s.listTeams)
	auth.POST("/teams", s.createTeam)
	auth.GET("/teams/:id", s.getTeam)
	auth.PUT("/teams/:id", s.updateTeam)
	auth.POST("/teams/:id/join", s.joinTeam)
	auth.POST("/teams/:id/leave", s.leaveTeam)
	auth.GET("/teams/:id/members", s.listTeamMembers)
	auth.PUT("/teams/:id/members/:user_id", s.updateTeamMember)
	auth.DELETE("/teams/:id/members/:user_id", s.removeTeamMember)
	auth.GET("/teams/:id/applications", s.listTeamApplications)
	auth.PUT("/teams/:id/applications/:application_id", s.reviewTeamApplication)
	auth.GET("/teams/:id/contests", s.listTeamContests)
	auth.POST("/teams/:id/contests", s.createTeamContest)
	auth.GET("/teams/:id/contests/:contest_id", s.getTeamContest)
	auth.POST("/teams/:id/contests/:contest_id/problems", s.addTeamContestProblem)
	auth.DELETE("/teams/:id/contests/:contest_id/problems/:problem_id", s.removeTeamContestProblem)
	auth.GET("/teams/:id/contests/:contest_id/submissions", s.listTeamContestSubmissions)
	auth.POST("/teams/:id/contests/:contest_id/submissions", s.createTeamContestSubmission)
	auth.GET("/teams/:id/contests/:contest_id/ranking", s.teamContestRanking)
	auth.GET("/teams/:id/problem-sets", s.listTeamProblemSets)
	auth.POST("/teams/:id/problem-sets", s.createTeamProblemSet)
	auth.GET("/teams/:id/problem-sets/:set_id", s.getTeamProblemSet)
	auth.POST("/teams/:id/problem-sets/:set_id/problems", s.addTeamProblemSetProblem)
	auth.DELETE("/teams/:id/problem-sets/:set_id/problems/:problem_id", s.removeTeamProblemSetProblem)
	auth.GET("/teams/:id/problem-sets/:set_id/submissions", s.listTeamProblemSetSubmissions)
	auth.POST("/teams/:id/problem-sets/:set_id/submissions", s.createTeamProblemSetSubmission)
	auth.GET("/teams/:id/problem-sets/:set_id/discussions", s.listTeamDiscussions)
	auth.POST("/teams/:id/problem-sets/:set_id/discussions", s.createTeamDiscussion)
	auth.GET("/courses", s.listCourses)
	auth.POST("/courses", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createCourse)
	auth.PUT("/courses/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.updateCourse)
	auth.DELETE("/courses/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.archiveCourse)
	auth.POST("/courses/:id/archive", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.archiveCourse)
	auth.POST("/courses/:id/classes", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createClass)
	auth.GET("/courses/:id/members", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.listCourseMembers)
	auth.POST("/courses/:id/members", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.addCourseMember)
	auth.DELETE("/courses/:id/members/:user_id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.removeCourseMember)
	auth.GET("/courses/preview", s.previewCourseByCode)
	auth.POST("/courses/join", s.joinCourseByCode)
	auth.POST("/courses/:id/leave", s.leaveCourse)
	auth.GET("/courses/:id/students", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.listCourseStudents)
	auth.GET("/classes", s.listClasses)
	auth.GET("/classes/join-preview", middleware.RequireRoles(models.RoleStudent), s.previewClassJoin)
	auth.GET("/me/classes", s.myClasses)
	auth.POST("/classes/join", middleware.RequireRoles(models.RoleStudent), s.joinClassByCode)
	auth.POST("/classes/:id/join", middleware.RequireRoles(models.RoleStudent), s.joinClass)
	auth.POST("/classes/:id/leave", middleware.RequireRoles(models.RoleStudent), s.leaveClass)
	auth.PUT("/classes/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.updateClass)
	auth.DELETE("/classes/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.archiveClass)
	auth.POST("/classes/:id/archive", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.archiveClass)
	auth.GET("/classes/:id/students", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.listClassStudents)
	auth.DELETE("/classes/:id/students/:user_id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.removeClassStudent)
	auth.POST("/classes/:id/students/import", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.importClassStudents)
	auth.POST("/problems", middleware.RateLimitUser(s.Redis, "problems:create", 20, time.Hour), s.createProblem)
	auth.PUT("/problem-test-uploads/:upload_id/:chunk_index", s.uploadProblemTestChunk)
	auth.POST("/problems/parse-markdown", middleware.RateLimitUser(s.Redis, "problems:parse-markdown", 30, time.Hour), s.parseMarkdownBatch)
	auth.POST("/problems/upload", middleware.RateLimitUser(s.Redis, "problems:upload", 20, time.Hour), s.uploadProblem)
	auth.PUT("/problems/:id", s.updateProblem)
	auth.GET("/problems/:id/tests", s.listProblemTests)
	auth.GET("/problems/:id/tests/download", s.downloadProblemTests)
	auth.GET("/problems/:id/tests/file/*file_path", s.downloadProblemTestFile)
	auth.DELETE("/problems/:id", s.deleteProblem)
	auth.GET("/prepared-problems", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.listPreparedProblems)
	auth.POST("/prepared-problems", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createPreparedProblem)
	auth.POST("/prepared-problems/upload", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.uploadPreparedProblem)
	auth.GET("/prepared-problems/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.getPreparedProblem)
	auth.PUT("/prepared-problems/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.updatePreparedProblem)
	auth.POST("/prepared-problems/:id/publish", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.publishPreparedProblem)
	auth.GET("/assignments", s.listAssignments)
	auth.GET("/assignments/:id", s.getAssignment)
	auth.POST("/assignments", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createAssignment)
	auth.GET("/assignments/:id/report", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.assignmentReport)
	auth.DELETE("/assignments/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.deleteAssignment)
	auth.GET("/exams", s.listExams)
	auth.GET("/exams/:id", s.getExam)
	auth.POST("/exams/:id/finish", middleware.RequireRoles(models.RoleStudent), s.finishExam)
	auth.POST("/exams", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createExam)
	auth.GET("/exams/:id/report/export", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.exportExamReport)
	auth.GET("/exams/:id/report", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.examReport)
	auth.GET("/exams/:id/ranking", s.examRanking)
	auth.DELETE("/exams/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.deleteExam)
	auth.POST("/exams/:id/submissions/:submission_id/judge", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.judgeManualExamSubmission)
	auth.PUT("/exams/:id/submissions/:submission_id/grade", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.gradeManualExamSubmission)
	auth.POST("/submissions", middleware.RateLimitUser(s.Redis, "submissions:create", 30, time.Minute), s.createSubmission)
	auth.GET("/submissions", s.listSubmissions)
	auth.GET("/submissions/:id", s.getSubmission)
	auth.GET("/submissions/:id/events", middleware.RateLimitUser(s.Redis, "submissions:events", 60, time.Minute), s.submissionEvents)
	// leaderboard disabled: auth.GET("/leaderboard", s.leaderboard)
	auth.GET("/plagiarism/jobs", middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.listPlagiarismJobs)
	auth.POST("/plagiarism/jobs", middleware.RateLimitUser(s.Redis, "plagiarism:create", 5, time.Hour), middleware.RequireRoles(models.RoleAdmin, models.RoleTeacher), s.createPlagiarismJob)
	auth.GET("/audit-logs", middleware.RequireRoles(models.RoleAdmin), s.listAuditLogs)
	auth.GET("/users", middleware.RequireRoles(models.RoleAdmin), s.listUsers)
	auth.POST("/users", middleware.RequireRoles(models.RoleAdmin), s.createUser)
	auth.GET("/users/:id", middleware.RequireRoles(models.RoleAdmin), s.getUser)
	auth.PUT("/users/:id", middleware.RequireRoles(models.RoleAdmin), s.updateUser)
	auth.DELETE("/users/:id", middleware.RequireRoles(models.RoleAdmin), s.deleteUser)
	auth.POST("/users/:id/reset-password", middleware.RequireRoles(models.RoleAdmin), s.resetUserPassword)
	auth.GET("/author-applications", middleware.RequireRoles(models.RoleAdmin), s.listAuthorApplications)
	auth.PUT("/author-applications/:id/review", middleware.RequireRoles(models.RoleAdmin), s.reviewAuthorApplication)
	auth.GET("/problem-authors", middleware.RequireRoles(models.RoleAdmin), s.listProblemAuthors)
	auth.DELETE("/problem-authors/:id", middleware.RequireRoles(models.RoleAdmin), s.removeProblemAuthor)
	auth.GET("/problem-reviews/mine", s.listMyProblemReviews)
	auth.GET("/problem-reviews", middleware.RequireRoles(models.RoleAdmin), s.listProblemReviews)
	auth.PUT("/problem-reviews/:id/review", middleware.RequireRoles(models.RoleAdmin), s.reviewProblem)
	auth.PUT("/problem-reviews/:id/withdraw", middleware.RequireRoles(models.RoleAdmin), s.withdrawProblem)
}
