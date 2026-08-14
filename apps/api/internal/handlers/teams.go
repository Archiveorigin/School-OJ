package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
	"school-oj/apps/api/internal/streams"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var teamSlugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,29}$`)

var (
	errTeamContestFrozen    = errors.New("team contest is frozen")
	errTeamContestDeleted   = errors.New("team contest was deleted")
	errTeamContestNoProblem = errors.New("team contest has no problems")
)

const defaultTeamContestAwardPercent = 10

type teamView struct {
	models.Team
	OwnerName        string          `json:"owner_name"`
	MemberCount      int64           `json:"member_count"`
	MyRole           models.TeamRole `json:"my_role,omitempty"`
	Joined           bool            `json:"joined"`
	ApplicationState string          `json:"application_status,omitempty"`
}

type teamMemberView struct {
	UserID    uint            `json:"user_id"`
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	AvatarURL string          `json:"avatar_url"`
	UserRole  models.Role     `json:"user_role"`
	TeamRole  models.TeamRole `json:"team_role"`
	JoinedAt  time.Time       `json:"joined_at"`
}

type teamApplicationView struct {
	models.TeamJoinApplication
	UserName string      `json:"user_name"`
	Email    string      `json:"email"`
	UserRole models.Role `json:"user_role"`
}

type teamProblemLinkView struct {
	models.TeamProblemSetProblem
	SubmissionStatus string     `json:"submission_status,omitempty"`
	SubmittedAt      *time.Time `json:"submitted_at,omitempty"`
	SubmissionCount  int64      `json:"submission_count"`
	AcceptedCount    int64      `json:"accepted_count"`
}

type teamContestView struct {
	models.TeamContest
	EndsAt       *time.Time `json:"ends_at,omitempty"`
	Status       string     `json:"status"`
	ProblemCount int64      `json:"problem_count"`
	CanEdit      bool       `json:"can_edit"`
	CanDelete    bool       `json:"can_delete"`
}

type teamProblemSetView struct {
	models.TeamProblemSet
	ProblemCount int64 `json:"problem_count"`
	CanEdit      bool  `json:"can_edit"`
	CanDelete    bool  `json:"can_delete"`
}

type teamContestInput struct {
	Title              string     `json:"title" binding:"required"`
	Description        string     `json:"description"`
	StartsAt           *time.Time `json:"starts_at"`
	DurationMinutes    int        `json:"duration_minutes"`
	ScoringRule        string     `json:"scoring_rule"`
	GoldAwardPercent   *int       `json:"gold_award_percent"`
	SilverAwardPercent *int       `json:"silver_award_percent"`
	BronzeAwardPercent *int       `json:"bronze_award_percent"`
}

type teamContestProblemView struct {
	models.TeamContestProblem
	SubmissionStatus string     `json:"submission_status,omitempty"`
	SubmittedAt      *time.Time `json:"submitted_at,omitempty"`
}

type teamSubmissionInput struct {
	ProblemID  uint   `json:"problem_id" binding:"required"`
	Language   string `json:"language" binding:"required"`
	SourceCode string `json:"source_code" binding:"required"`
}

type teamContentMutationError struct {
	status  int
	message string
}

func (err *teamContentMutationError) Error() string {
	return err.message
}

type teamContestRankingCell struct {
	ProblemID      uint       `json:"problem_id"`
	Status         string     `json:"status"`
	Attempts       int        `json:"attempts"`
	WrongAttempts  int        `json:"wrong_attempts"`
	BestScore      int        `json:"best_score"`
	MaxScore       int        `json:"max_score"`
	ElapsedMinutes int        `json:"elapsed_minutes"`
	SolvedAt       *time.Time `json:"solved_at,omitempty"`
	Fastest        bool       `json:"fastest"`
}

type teamContestRankingRow struct {
	UserID          uint                     `json:"user_id"`
	Name            string                   `json:"name"`
	Solved          int                      `json:"solved"`
	TotalScore      int                      `json:"total_score"`
	MaxScore        int                      `json:"max_score"`
	PenaltyMinutes  int                      `json:"penalty_minutes"`
	SubmissionCount int                      `json:"submission_count"`
	LastSubmission  *time.Time               `json:"last_submission,omitempty"`
	Problems        []teamContestRankingCell `json:"problems"`
}

type teamContestCellAggregate struct {
	UserID          uint
	ProblemID       uint
	Attempts        int
	SubmissionCount int
	WrongAttempts   int
	BestScore       int
	SolvedAt        *time.Time
	LastSubmission  time.Time
	LatestStatus    string
}

type teamDiscussionView struct {
	models.TeamDiscussion
	AuthorName   string `json:"author_name"`
	AuthorAvatar string `json:"author_avatar"`
}

func (s Server) listTeams(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	scope := strings.ToLower(strings.TrimSpace(c.DefaultQuery("scope", "discover")))
	var teams []models.Team
	query := s.DB.Model(&models.Team{}).Order("teams.updated_at desc")
	if scope == "mine" {
		query = query.Where("teams.id IN (?)", s.DB.Model(&models.TeamMembership{}).Select("team_id").Where("user_id = ?", user.ID))
	} else {
		query = query.Where("teams.visibility = ?", "public")
	}
	keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("lower(teams.name) LIKE ? OR lower(teams.slug) LIKE ? OR lower(teams.description) LIKE ?", like, like, like)
	}
	if err := query.Find(&teams).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s.teamViews(teams, user.ID))
}

func (s Server) teamViews(teams []models.Team, userID uint) []teamView {
	views := make([]teamView, 0, len(teams))
	for _, team := range teams {
		var owner models.User
		_ = s.DB.Select("id", "name").First(&owner, team.OwnerID).Error
		var memberCount int64
		s.DB.Model(&models.TeamMembership{}).Where("team_id = ?", team.ID).Count(&memberCount)
		membership, joined := s.teamMembership(team.ID, userID)
		applicationState := ""
		if !joined {
			var application models.TeamJoinApplication
			if err := s.DB.Where("team_id = ? AND user_id = ?", team.ID, userID).Order("id desc").First(&application).Error; err == nil {
				applicationState = application.Status
			}
		}
		views = append(views, teamView{
			Team:             team,
			OwnerName:        owner.Name,
			MemberCount:      memberCount,
			MyRole:           membership.Role,
			Joined:           joined,
			ApplicationState: applicationState,
		})
	}
	return views
}

func (s Server) createTeam(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	var req struct {
		Name              string `json:"name" binding:"required"`
		Slug              string `json:"slug" binding:"required"`
		Visibility        string `json:"visibility"`
		JoinMode          string `json:"join_mode"`
		ContestPermission string `json:"contest_permission"`
		Description       string `json:"description"`
		Announcement      string `json:"announcement"`
		IconURL           string `json:"icon_url"`
	}
	if !bind(c, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if len([]rune(req.Name)) > 120 || !teamSlugPattern.MatchString(req.Slug) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队链接名须以小写字母开头，只能包含小写字母、数字和连字符，长度为 2-30 个字符"})
		return
	}
	if len([]rune(req.Description)) > 140 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队简介不能超过 140 个字符"})
		return
	}
	if !validTeamIconURL(req.IconURL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队图标格式无效或超过 2MB"})
		return
	}
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if req.JoinMode == "" {
		req.JoinMode = "application"
	}
	if req.ContestPermission == "" {
		req.ContestPermission = "admin"
	}
	if !validTeamSetting(req.Visibility, []string{"private", "public"}) ||
		!validTeamSetting(req.JoinMode, []string{"invitation", "application", "open"}) ||
		!validTeamSetting(req.ContestPermission, []string{"all", "admin", "owner"}) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队设置值无效"})
		return
	}
	team := models.Team{
		Name:              req.Name,
		Slug:              req.Slug,
		OwnerID:           user.ID,
		Visibility:        req.Visibility,
		JoinMode:          req.JoinMode,
		ContestPermission: req.ContestPermission,
		JoinCode:          newTeamJoinCode(),
		Description:       strings.TrimSpace(req.Description),
		Announcement:      strings.TrimSpace(req.Announcement),
		IconURL:           strings.TrimSpace(req.IconURL),
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&team).Error; err != nil {
			return err
		}
		return tx.Create(&models.TeamMembership{TeamID: team.ID, UserID: user.ID, Role: models.TeamRoleOwner}).Error
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队链接名已存在或数据无效"})
		return
	}
	services.Audit(c, s.DB, "team.create", "team", team.ID, datatypes.JSONMap{"slug": team.Slug})
	c.JSON(http.StatusCreated, s.teamViews([]models.Team{team}, user.ID)[0])
}

func (s Server) getTeam(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByParam(c)
	if !ok {
		return
	}
	_, joined := s.teamMembership(team.ID, user.ID)
	if team.Visibility != "public" && !joined && user.Role != models.RoleAdmin {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}
	c.JSON(http.StatusOK, s.teamViews([]models.Team{team}, user.ID)[0])
}

func (s Server) updateTeam(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByIDParam(c)
	if !ok {
		return
	}
	membership, joined := s.teamMembership(team.ID, user.ID)
	if user.Role != models.RoleAdmin && (!joined || (membership.Role != models.TeamRoleOwner && membership.Role != models.TeamRoleAdmin)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var req struct {
		Name              string `json:"name"`
		Visibility        string `json:"visibility"`
		JoinMode          string `json:"join_mode"`
		ContestPermission string `json:"contest_permission"`
		Description       string `json:"description"`
		Announcement      string `json:"announcement"`
		IconURL           string `json:"icon_url"`
	}
	if !bind(c, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" || len([]rune(req.Name)) > 120 || len([]rune(req.Description)) > 140 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队名称或简介过长"})
		return
	}
	if !validTeamIconURL(req.IconURL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "团队图标格式无效或超过 2MB"})
		return
	}
	updates := map[string]any{
		"name":         strings.TrimSpace(req.Name),
		"description":  strings.TrimSpace(req.Description),
		"announcement": strings.TrimSpace(req.Announcement),
		"icon_url":     strings.TrimSpace(req.IconURL),
	}
	if membership.Role == models.TeamRoleOwner || user.Role == models.RoleAdmin {
		if !validTeamSetting(req.Visibility, []string{"private", "public"}) ||
			!validTeamSetting(req.JoinMode, []string{"invitation", "application", "open"}) ||
			!validTeamSetting(req.ContestPermission, []string{"all", "admin", "owner"}) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "团队设置值无效"})
			return
		}
		updates["visibility"] = req.Visibility
		updates["join_mode"] = req.JoinMode
		updates["contest_permission"] = req.ContestPermission
	}
	if err := s.DB.Model(&team).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.DB.First(&team, team.ID)
	services.Audit(c, s.DB, "team.update", "team", team.ID, nil)
	c.JSON(http.StatusOK, s.teamViews([]models.Team{team}, user.ID)[0])
}

func validTeamIconURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	if len(value) > 3<<20 {
		return false
	}
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "data:image/png;base64,") ||
		strings.HasPrefix(lower, "data:image/jpeg;base64,") ||
		strings.HasPrefix(lower, "data:image/webp;base64,") ||
		(len(value) <= 2048 && (strings.HasPrefix(lower, "https://") || strings.HasPrefix(value, "/")))
}

func (s Server) joinTeam(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByIDParam(c)
	if !ok {
		return
	}
	if _, joined := s.teamMembership(team.ID, user.ID); joined {
		c.JSON(http.StatusOK, gin.H{"joined": true})
		return
	}
	var req struct {
		JoinCode string `json:"join_code"`
		Message  string `json:"message"`
	}
	if !bind(c, &req) {
		return
	}
	switch team.JoinMode {
	case "open":
		if err := s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: user.ID, Role: models.TeamRoleMember}).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		services.Audit(c, s.DB, "team.join", "team", team.ID, nil)
		c.JSON(http.StatusCreated, gin.H{"joined": true})
	case "invitation":
		if team.JoinCode == "" || !strings.EqualFold(strings.TrimSpace(req.JoinCode), team.JoinCode) {
			c.JSON(http.StatusForbidden, gin.H{"error": "团队邀请码无效"})
			return
		}
		if err := s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: user.ID, Role: models.TeamRoleMember}).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		services.Audit(c, s.DB, "team.join.invitation", "team", team.ID, nil)
		c.JSON(http.StatusCreated, gin.H{"joined": true})
	default:
		application := models.TeamJoinApplication{TeamID: team.ID, UserID: user.ID, Message: strings.TrimSpace(req.Message), Status: "pending"}
		if err := s.DB.Create(&application).Error; err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "已有待处理的加入申请"})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"joined": false, "application_status": "pending"})
	}
}

func (s Server) leaveTeam(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByIDParam(c)
	if !ok {
		return
	}
	membership, joined := s.teamMembership(team.ID, user.ID)
	if !joined {
		c.JSON(http.StatusNotFound, gin.H{"error": "team membership not found"})
		return
	}
	if membership.Role == models.TeamRoleOwner {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建者不能退出团队，请先转让团队"})
		return
	}
	if err := s.DB.Delete(&membership).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "team.leave", "team", team.ID, nil)
	c.JSON(http.StatusOK, gin.H{"left": true})
}

func (s Server) listTeamMembers(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByIDParam(c)
	if !ok || !s.requireTeamMember(c, user, team) {
		return
	}
	var rows []struct {
		UserID    uint
		Name      string
		Email     string
		AvatarURL string
		UserRole  models.Role
		TeamRole  models.TeamRole
		JoinedAt  time.Time
	}
	s.DB.Table("team_memberships").
		Select("users.id AS user_id, users.name, users.email, users.avatar_url, users.role AS user_role, team_memberships.role AS team_role, team_memberships.created_at AS joined_at").
		Joins("JOIN users ON users.id = team_memberships.user_id").
		Where("team_memberships.team_id = ? AND users.account_deleted = false", team.ID).
		Order("CASE team_memberships.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, team_memberships.created_at").
		Scan(&rows)
	result := make([]teamMemberView, 0, len(rows))
	for _, row := range rows {
		result = append(result, teamMemberView(row))
	}
	c.JSON(http.StatusOK, result)
}

func (s Server) updateTeamMember(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByIDParam(c)
	if !ok || !s.requireTeamManager(c, user, team) {
		return
	}
	targetID, ok := idParam(c, "user_id")
	if !ok {
		return
	}
	var req struct {
		Role models.TeamRole `json:"role" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	if req.Role != models.TeamRoleAdmin && req.Role != models.TeamRoleMember {
		c.JSON(http.StatusBadRequest, gin.H{"error": "成员职级只能设为管理员或团员"})
		return
	}
	actor, _ := s.teamMembership(team.ID, user.ID)
	target, found := s.teamMembership(team.ID, targetID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "team member not found"})
		return
	}
	if target.Role == models.TeamRoleOwner || (actor.Role != models.TeamRoleOwner && user.Role != models.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有创建者可以调整成员职级"})
		return
	}
	if err := s.DB.Model(&target).Update("role", req.Role).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "team.member.role", "team", team.ID, datatypes.JSONMap{"user_id": targetID, "role": req.Role})
	c.JSON(http.StatusOK, gin.H{"updated": true})
}

func (s Server) removeTeamMember(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByIDParam(c)
	if !ok || !s.requireTeamManager(c, user, team) {
		return
	}
	targetID, ok := idParam(c, "user_id")
	if !ok {
		return
	}
	actor, _ := s.teamMembership(team.ID, user.ID)
	target, found := s.teamMembership(team.ID, targetID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "team member not found"})
		return
	}
	if target.Role == models.TeamRoleOwner || (target.Role == models.TeamRoleAdmin && actor.Role != models.TeamRoleOwner && user.Role != models.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限移除该成员"})
		return
	}
	if err := s.DB.Delete(&target).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "team.member.remove", "team", team.ID, datatypes.JSONMap{"user_id": targetID})
	c.JSON(http.StatusOK, gin.H{"removed": true})
}

func (s Server) listTeamApplications(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByIDParam(c)
	if !ok || !s.requireTeamManager(c, user, team) {
		return
	}
	var rows []teamApplicationView
	s.DB.Table("team_join_applications").
		Select("team_join_applications.*, users.name AS user_name, users.email, users.role AS user_role").
		Joins("JOIN users ON users.id = team_join_applications.user_id").
		Where("team_join_applications.team_id = ? AND team_join_applications.status = 'pending'", team.ID).
		Order("team_join_applications.created_at").
		Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

func (s Server) reviewTeamApplication(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByIDParam(c)
	if !ok || !s.requireTeamManager(c, user, team) {
		return
	}
	applicationID, ok := idParam(c, "application_id")
	if !ok {
		return
	}
	var req struct {
		Action string `json:"action" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	if req.Action != "approve" && req.Action != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be approve or reject"})
		return
	}
	var application models.TeamJoinApplication
	if err := s.DB.Where("id = ? AND team_id = ? AND status = 'pending'", applicationID, team.ID).First(&application).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	now := time.Now()
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if req.Action == "approve" {
			membership := models.TeamMembership{TeamID: team.ID, UserID: application.UserID, Role: models.TeamRoleMember}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&membership).Error; err != nil {
				return err
			}
		}
		return tx.Model(&application).Updates(map[string]any{"status": map[bool]string{true: "approved", false: "rejected"}[req.Action == "approve"], "reviewed_by": user.ID, "reviewed_at": now}).Error
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reviewed": true})
}

func (s Server) listTeamContests(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByIDParam(c)
	if !ok || !s.requireTeamMember(c, user, team) {
		return
	}
	canManage := s.canManageTeam(user, team)
	var items []models.TeamContest
	s.DB.Where("team_id = ? AND deleted_at IS NULL", team.ID).Order("starts_at desc nulls last, id desc").Find(&items)
	views := make([]teamContestView, 0, len(items))
	for _, item := range items {
		var count int64
		s.DB.Model(&models.TeamContestProblem{}).Where("contest_id = ?", item.ID).Count(&count)
		endsAt, status := teamContestWindow(item, time.Now())
		views = append(views, teamContestView{
			TeamContest:  item,
			EndsAt:       endsAt,
			Status:       status,
			ProblemCount: count,
			CanEdit:      canManage && status == models.TeamContestDraft,
			CanDelete:    canManage,
		})
	}
	c.JSON(http.StatusOK, views)
}

func (s Server) createTeamContest(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByIDParam(c)
	if !ok || !s.requireTeamContentPermission(c, user, team) {
		return
	}
	var req teamContestInput
	if !bind(c, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "比赛标题不能为空"})
		return
	}
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 120
	}
	req.ScoringRule = strings.ToLower(strings.TrimSpace(req.ScoringRule))
	if req.ScoringRule == "" {
		req.ScoringRule = "penalty"
	}
	if req.ScoringRule != "score" && req.ScoringRule != "penalty" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评分规则必须为 score 或 penalty"})
		return
	}
	goldPercent, silverPercent, bronzePercent, err := resolveTeamContestAwardPercentages(
		req.GoldAwardPercent,
		req.SilverAwardPercent,
		req.BronzeAwardPercent,
		defaultTeamContestAwardPercent,
		defaultTeamContestAwardPercent,
		defaultTeamContestAwardPercent,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := models.TeamContest{
		TeamID:             team.ID,
		Title:              req.Title,
		Description:        strings.TrimSpace(req.Description),
		StartsAt:           req.StartsAt,
		DurationMinutes:    req.DurationMinutes,
		ScoringRule:        req.ScoringRule,
		GoldAwardPercent:   &goldPercent,
		SilverAwardPercent: &silverPercent,
		BronzeAwardPercent: &bronzePercent,
		State:              models.TeamContestDraft,
		CreatedBy:          user.ID,
	}
	if err := s.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "team.contest.create", "team_contest", item.ID, datatypes.JSONMap{"team_id": team.ID})
	c.JSON(http.StatusCreated, item)
}

func (s Server) updateTeamContest(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	contest, team, ok := s.teamContestByParams(c)
	if !ok || !s.requireTeamManager(c, user, team) {
		return
	}
	if !requireDraftTeamContest(c, contest) {
		return
	}
	var req teamContestInput
	if !bind(c, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.ScoringRule = strings.ToLower(strings.TrimSpace(req.ScoringRule))
	if req.Title == "" || req.DurationMinutes <= 0 || (req.ScoringRule != "score" && req.ScoringRule != "penalty") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "比赛时长和评分规则无效"})
		return
	}
	goldPercent, silverPercent, bronzePercent, err := resolveTeamContestAwardPercentages(
		req.GoldAwardPercent,
		req.SilverAwardPercent,
		req.BronzeAwardPercent,
		teamContestAwardPercentValue(contest.GoldAwardPercent),
		teamContestAwardPercentValue(contest.SilverAwardPercent),
		teamContestAwardPercentValue(contest.BronzeAwardPercent),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]any{
		"title":                req.Title,
		"description":          strings.TrimSpace(req.Description),
		"starts_at":            req.StartsAt,
		"duration_minutes":     req.DurationMinutes,
		"scoring_rule":         req.ScoringRule,
		"gold_award_percent":   goldPercent,
		"silver_award_percent": silverPercent,
		"bronze_award_percent": bronzePercent,
	}
	result := s.DB.Model(&models.TeamContest{}).Where("id = ? AND state = ? AND deleted_at IS NULL", contest.ID, models.TeamContestDraft).Updates(updates)
	if result.Error != nil || result.RowsAffected != 1 {
		c.JSON(http.StatusConflict, gin.H{"error": "比赛已发布，设置已冻结"})
		return
	}
	_ = s.DB.Where("deleted_at IS NULL").First(&contest, contest.ID).Error
	services.Audit(c, s.DB, "team.contest.update", "team_contest", contest.ID, datatypes.JSONMap{"team_id": team.ID})
	c.JSON(http.StatusOK, contest)
}

func (s Server) deleteTeamContest(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	contest, team, ok := s.teamContestByParams(c)
	if !ok || !s.requireTeamManager(c, user, team) {
		return
	}
	now := time.Now()
	result := s.DB.Model(&models.TeamContest{}).
		Where("id = ? AND team_id = ? AND deleted_at IS NULL", contest.ID, team.ID).
		Update("deleted_at", now)
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected != 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contest not found"})
		return
	}
	services.Audit(c, s.DB, "team.contest.delete", "team_contest", contest.ID, datatypes.JSONMap{"team_id": team.ID, "state": contest.State})
	c.JSON(http.StatusOK, gin.H{"deleted": true, "contest_id": contest.ID})
}

func (s Server) publishTeamContest(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	contest, team, ok := s.teamContestByParams(c)
	if !ok || !s.requireTeamContentPermission(c, user, team) {
		return
	}
	now := time.Now()
	if contest.State != models.TeamContestDraft {
		c.JSON(http.StatusConflict, gin.H{"error": "比赛已发布，题目和评分规则已冻结"})
		return
	}
	if contest.StartsAt == nil || !contest.StartsAt.After(now) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "发布前必须设置晚于当前时间的开始时间"})
		return
	}
	var problemCount int64
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.TeamContest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("deleted_at IS NULL").First(&locked, contest.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errTeamContestDeleted
			}
			return err
		}
		if locked.State != models.TeamContestDraft {
			return errTeamContestFrozen
		}
		if locked.StartsAt == nil || !locked.StartsAt.After(now) {
			return gorm.ErrInvalidData
		}
		contest = locked
		if err := tx.Model(&models.TeamContestProblem{}).Where("contest_id = ?", contest.ID).Count(&problemCount).Error; err != nil {
			return err
		}
		if problemCount == 0 {
			return errTeamContestNoProblem
		}
		if err := tx.Exec(`
INSERT INTO team_contest_participants(contest_id, user_id)
SELECT ?, memberships.user_id
FROM team_memberships memberships
JOIN users ON users.id = memberships.user_id
WHERE memberships.team_id = ?
  AND users.account_deleted = false
ON CONFLICT (contest_id, user_id) DO NOTHING`, contest.ID, team.ID).Error; err != nil {
			return err
		}
		return tx.Model(&locked).Updates(map[string]any{"state": models.TeamContestPublished, "published_at": now}).Error
	}); err != nil {
		if status, message, ok := teamContestPublishErrorResponse(err); ok {
			c.JSON(status, gin.H{"error": message})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "比赛状态已变化，请刷新后重试"})
		return
	}
	services.Audit(c, s.DB, "team.contest.publish", "team_contest", contest.ID, datatypes.JSONMap{"team_id": team.ID, "problem_count": problemCount, "scoring_rule": contest.ScoringRule})
	contest.State = models.TeamContestPublished
	contest.PublishedAt = &now
	c.JSON(http.StatusOK, contest)
}

func (s Server) getTeamContest(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	contest, team, ok := s.teamContestByParams(c)
	if !ok || !s.requireTeamMember(c, user, team) {
		return
	}
	endsAt, status := teamContestWindow(contest, time.Now())
	canOrganize := s.canTeamContentPermission(user, team)
	canManage := s.canManageTeam(user, team)
	if (status == models.TeamContestDraft || status == models.TeamContestPublished) && !canOrganize && !canManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "比赛尚未开始"})
		return
	}
	var links []models.TeamContestProblem
	s.DB.Preload("Problem").Where("contest_id = ?", contest.ID).Order("sort_order, id").Find(&links)
	links = filterActiveTeamContestProblemLinks(links)
	normalizeTeamContestProblemLabels(links)
	views := make([]teamContestProblemView, 0, len(links))
	var participantCount int64
	s.DB.Model(&models.TeamContestParticipant{}).Where("contest_id = ? AND user_id = ?", contest.ID, user.ID).Count(&participantCount)
	for _, link := range links {
		var latest models.Submission
		view := teamContestProblemView{TeamContestProblem: link}
		if err := s.DB.Where("user_id = ? AND team_contest_id = ? AND problem_id = ?", user.ID, contest.ID, link.ProblemID).Order("created_at desc").First(&latest).Error; err == nil {
			view.SubmissionStatus = string(latest.Status)
			value := latest.CreatedAt
			view.SubmittedAt = &value
			var accepted int64
			s.DB.Model(&models.Submission{}).Where("user_id = ? AND team_contest_id = ? AND problem_id = ? AND status = ?", user.ID, contest.ID, link.ProblemID, models.StatusAccepted).Count(&accepted)
			if accepted > 0 {
				view.SubmissionStatus = string(models.StatusAccepted)
			}
		}
		views = append(views, view)
	}
	c.JSON(http.StatusOK, gin.H{
		"contest": teamContestView{
			TeamContest:  contest,
			EndsAt:       endsAt,
			Status:       status,
			ProblemCount: int64(len(views)),
			CanEdit:      canManage && status == models.TeamContestDraft,
			CanDelete:    canManage,
		},
		"team":           s.teamViews([]models.Team{team}, user.ID)[0],
		"problems":       views,
		"can_organize":   canOrganize,
		"can_submit":     status == models.TeamContestRunning && participantCount > 0,
		"is_participant": participantCount > 0,
		"can_edit":       canManage && status == models.TeamContestDraft,
		"can_delete":     canManage,
		"can_publish":    canOrganize && status == models.TeamContestDraft,
	})
}

func (s Server) addTeamContestProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	contest, team, ok := s.teamContestByParams(c)
	if !ok || !s.requireTeamContentPermission(c, user, team) {
		return
	}
	if !requireDraftTeamContest(c, contest) {
		return
	}
	var req struct {
		ProblemID   uint   `json:"problem_id"`
		ProblemCode string `json:"problem_code"`
		Label       string `json:"label"`
	}
	if !bind(c, &req) {
		return
	}
	problem, found := s.teamScopedProblem(c, user, team, req.ProblemID, req.ProblemCode)
	if !found {
		return
	}
	link := models.TeamContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Label: strings.TrimSpace(req.Label)}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.TeamContest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("deleted_at IS NULL").First(&locked, contest.ID).Error; err != nil {
			return err
		}
		if locked.State != models.TeamContestDraft {
			return errTeamContestFrozen
		}
		var count int64
		if err := tx.Model(&models.TeamContestProblem{}).Where("contest_id = ?", contest.ID).Count(&count).Error; err != nil {
			return err
		}
		link.SortOrder = int(count)
		return tx.Create(&link).Error
	}); err != nil {
		if errors.Is(err, errTeamContestFrozen) {
			c.JSON(http.StatusConflict, gin.H{"error": "比赛发布后题目已冻结"})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "题目已在该比赛中"})
		return
	}
	c.JSON(http.StatusCreated, link)
}

func (s Server) removeTeamContestProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	contest, team, ok := s.teamContestByParams(c)
	if !ok || !s.requireTeamContentPermission(c, user, team) {
		return
	}
	if !requireDraftTeamContest(c, contest) {
		return
	}
	problemID, ok := idParam(c, "problem_id")
	if !ok {
		return
	}
	var removed bool
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.TeamContest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("deleted_at IS NULL").First(&locked, contest.ID).Error; err != nil {
			return err
		}
		if locked.State != models.TeamContestDraft {
			return errTeamContestFrozen
		}
		result := tx.Where("contest_id = ? AND problem_id = ?", contest.ID, problemID).Delete(&models.TeamContestProblem{})
		removed = result.RowsAffected > 0
		return result.Error
	})
	if err != nil {
		if errors.Is(err, errTeamContestFrozen) {
			c.JSON(http.StatusConflict, gin.H{"error": "比赛发布后题目已冻结"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"removed": removed})
}

func (s Server) listTeamContestSubmissions(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	contest, team, ok := s.teamContestByParams(c)
	if !ok || !s.requireTeamMember(c, user, team) {
		return
	}
	var items []models.Submission
	s.DB.Where("user_id = ? AND team_contest_id = ?", user.ID, contest.ID).Order("id desc").Limit(200).Find(&items)
	c.JSON(http.StatusOK, s.submissionListViews(items))
}

func (s Server) createTeamContestSubmission(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	contest, team, ok := s.teamContestByParams(c)
	if !ok || !s.requireTeamMember(c, user, team) {
		return
	}
	_, status := teamContestWindow(contest, time.Now())
	if status != models.TeamContestRunning {
		message := "比赛已结束"
		if status == models.TeamContestDraft || status == models.TeamContestPublished {
			message = "比赛尚未开始"
		}
		c.JSON(http.StatusForbidden, gin.H{"error": message})
		return
	}
	var participantCount int64
	s.DB.Model(&models.TeamContestParticipant{}).Where("contest_id = ? AND user_id = ?", contest.ID, user.ID).Count(&participantCount)
	if participantCount == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "你不在比赛发布时冻结的参赛名单中"})
		return
	}
	var req teamSubmissionInput
	if !bind(c, &req) {
		return
	}
	var count int64
	s.DB.Model(&models.TeamContestProblem{}).Where("contest_id = ? AND problem_id = ?", contest.ID, req.ProblemID).Count(&count)
	if count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "题目不属于当前比赛"})
		return
	}
	s.createScopedTeamSubmission(c, user, req, &contest.ID, nil)
}

func (s Server) teamContestRanking(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	contest, team, ok := s.teamContestByParams(c)
	if !ok || !s.requireTeamMember(c, user, team) {
		return
	}
	_, contestStatus := teamContestWindow(contest, time.Now())
	canOrganize := s.canTeamContentPermission(user, team)
	canManage := s.canManageTeam(user, team)
	if (contestStatus == models.TeamContestDraft || contestStatus == models.TeamContestPublished) && !canOrganize && !canManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "比赛尚未开始"})
		return
	}
	var links []models.TeamContestProblem
	s.DB.Preload("Problem").Where("contest_id = ?", contest.ID).Order("sort_order, id").Find(&links)
	links = filterActiveTeamContestProblemLinks(links)
	normalizeTeamContestProblemLabels(links)
	var userIDs []uint
	if contest.State == models.TeamContestDraft {
		s.DB.Model(&models.TeamMembership{}).Where("team_id = ?", team.ID).Pluck("user_id", &userIDs)
	} else {
		s.DB.Model(&models.TeamContestParticipant{}).Where("contest_id = ?", contest.ID).Pluck("user_id", &userIDs)
	}
	var users []models.User
	if len(userIDs) > 0 {
		s.DB.Where("id IN ? AND account_deleted = false", userIDs).Order("lower(name), id").Find(&users)
	}
	users = activeTeamContestUsers(users)
	aggregates, err := s.loadTeamContestCellAggregates(contest.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	contestStart := contest.CreatedAt
	if contest.StartsAt != nil {
		contestStart = *contest.StartsAt
	}
	rows := buildTeamContestRanking(users, links, aggregates, contestStart, contest.ScoringRule)

	problems := make([]gin.H, 0, len(links))
	for _, link := range links {
		problems = append(problems, gin.H{"problem_id": link.ProblemID, "label": link.Label, "title": link.Problem.Title, "score": 100})
	}
	c.JSON(http.StatusOK, gin.H{
		"contest":           contest,
		"scoring_rule":      contest.ScoringRule,
		"participant_count": len(rows),
		"problems":          problems,
		"rows":              rows,
	})
}

func (s Server) loadTeamContestCellAggregates(contestID uint) ([]teamContestCellAggregate, error) {
	var rows []teamContestCellAggregate
	err := s.DB.Raw(`
WITH scoped AS (
  SELECT user_id, problem_id, status, score, created_at, id
  FROM submissions
  WHERE team_contest_id = ?
), first_accept AS (
  SELECT DISTINCT ON (user_id, problem_id)
         user_id, problem_id, created_at AS solved_at, id AS solved_id
  FROM scoped
  WHERE status = ?
  ORDER BY user_id, problem_id, created_at, id
), latest AS (
  SELECT DISTINCT ON (user_id, problem_id) user_id, problem_id, status
  FROM scoped
  ORDER BY user_id, problem_id, created_at DESC, id DESC
)
SELECT scoped.user_id,
       scoped.problem_id,
       COUNT(*) FILTER (
         WHERE scoped.status <> ?
           AND (
             first_accept.solved_at IS NULL
             OR (scoped.created_at, scoped.id) <= (first_accept.solved_at, first_accept.solved_id)
           )
       )::integer AS attempts,
       COUNT(*)::integer AS submission_count,
       COUNT(*) FILTER (
         WHERE first_accept.solved_at IS NOT NULL
           AND (scoped.created_at, scoped.id) < (first_accept.solved_at, first_accept.solved_id)
           AND scoped.status NOT IN (?, ?, ?, ?, ?)
       )::integer AS wrong_attempts,
       COALESCE(MAX(scoped.score), 0)::integer AS best_score,
       first_accept.solved_at,
       MAX(scoped.created_at) AS last_submission,
       CASE WHEN first_accept.solved_at IS NOT NULL THEN ? ELSE latest.status END AS latest_status
FROM scoped
LEFT JOIN first_accept USING (user_id, problem_id)
JOIN latest USING (user_id, problem_id)
GROUP BY scoped.user_id, scoped.problem_id, first_accept.solved_at, first_accept.solved_id, latest.status`,
		contestID,
		models.StatusAccepted,
		models.StatusSystemError,
		models.StatusQueued,
		models.StatusRunning,
		models.StatusPendingReview,
		models.StatusAccepted,
		models.StatusSystemError,
		models.StatusAccepted,
	).Scan(&rows).Error
	return rows, err
}

func buildTeamContestRanking(users []models.User, links []models.TeamContestProblem, aggregates []teamContestCellAggregate, contestStart time.Time, scoringRule string) []teamContestRankingRow {
	byUserProblem := make(map[[2]uint]teamContestCellAggregate, len(aggregates))
	for _, aggregate := range aggregates {
		byUserProblem[[2]uint{aggregate.UserID, aggregate.ProblemID}] = aggregate
	}
	rows := make([]teamContestRankingRow, 0, len(users))
	for _, member := range users {
		row := teamContestRankingRow{UserID: member.ID, Name: member.Name, MaxScore: len(links) * 100, Problems: make([]teamContestRankingCell, 0, len(links))}
		for _, link := range links {
			cell := teamContestRankingCell{ProblemID: link.ProblemID, MaxScore: 100}
			if aggregate, ok := byUserProblem[[2]uint{member.ID, link.ProblemID}]; ok {
				cell.Status = aggregate.LatestStatus
				cell.Attempts = aggregate.Attempts
				cell.WrongAttempts = aggregate.WrongAttempts
				cell.BestScore = aggregate.BestScore
				cell.SolvedAt = aggregate.SolvedAt
				cell.ElapsedMinutes = elapsedContestMinutes(contestStart, aggregate.LastSubmission)
				row.SubmissionCount += aggregate.SubmissionCount
				last := aggregate.LastSubmission
				if row.LastSubmission == nil || last.After(*row.LastSubmission) {
					row.LastSubmission = &last
				}
				if aggregate.SolvedAt != nil {
					cell.ElapsedMinutes = elapsedContestMinutes(contestStart, *aggregate.SolvedAt)
					row.Solved++
					row.PenaltyMinutes += cell.ElapsedMinutes + cell.WrongAttempts*20
				}
				row.TotalScore += cell.BestScore
			}
			row.Problems = append(row.Problems, cell)
		}
		rows = append(rows, row)
	}
	fastestByProblem := map[uint]time.Time{}
	for _, row := range rows {
		for _, cell := range row.Problems {
			if cell.SolvedAt == nil {
				continue
			}
			if fastest, ok := fastestByProblem[cell.ProblemID]; !ok || cell.SolvedAt.Before(fastest) {
				fastestByProblem[cell.ProblemID] = *cell.SolvedAt
			}
		}
	}
	for rowIndex := range rows {
		for cellIndex := range rows[rowIndex].Problems {
			cell := &rows[rowIndex].Problems[cellIndex]
			if cell.SolvedAt != nil && cell.SolvedAt.Equal(fastestByProblem[cell.ProblemID]) {
				cell.Fastest = true
			}
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if scoringRule == "score" {
			if rows[i].TotalScore != rows[j].TotalScore {
				return rows[i].TotalScore > rows[j].TotalScore
			}
			if rows[i].Solved != rows[j].Solved {
				return rows[i].Solved > rows[j].Solved
			}
		} else {
			if rows[i].Solved != rows[j].Solved {
				return rows[i].Solved > rows[j].Solved
			}
			if rows[i].PenaltyMinutes != rows[j].PenaltyMinutes {
				return rows[i].PenaltyMinutes < rows[j].PenaltyMinutes
			}
		}
		return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
	})
	return rows
}

func activeTeamContestUsers(users []models.User) []models.User {
	active := make([]models.User, 0, len(users))
	for _, user := range users {
		if !user.AccountDeleted {
			active = append(active, user)
		}
	}
	return active
}

func elapsedContestMinutes(start, submittedAt time.Time) int {
	if submittedAt.Before(start) {
		return 0
	}
	return int(submittedAt.Sub(start) / time.Minute)
}

func (s Server) listTeamProblemSets(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByIDParam(c)
	if !ok || !s.requireTeamMember(c, user, team) {
		return
	}
	canManage := s.canManageTeam(user, team)
	var items []models.TeamProblemSet
	s.DB.Where("team_id = ? AND deleted_at IS NULL", team.ID).Order("updated_at desc").Find(&items)
	views := make([]teamProblemSetView, 0, len(items))
	for _, item := range items {
		var count int64
		s.DB.Model(&models.TeamProblemSetProblem{}).Where("problem_set_id = ?", item.ID).Count(&count)
		views = append(views, teamProblemSetView{TeamProblemSet: item, ProblemCount: count, CanEdit: canManage, CanDelete: canManage})
	}
	c.JSON(http.StatusOK, views)
}

func (s Server) createTeamProblemSet(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByIDParam(c)
	if !ok || !s.requireTeamContentPermission(c, user, team) {
		return
	}
	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}
	if !bind(c, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "题单标题不能为空"})
		return
	}
	item := models.TeamProblemSet{TeamID: team.ID, Title: req.Title, Description: strings.TrimSpace(req.Description), CreatedBy: user.ID}
	if err := s.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	services.Audit(c, s.DB, "team.problem_set.create", "team_problem_set", item.ID, datatypes.JSONMap{"team_id": team.ID})
	c.JSON(http.StatusCreated, item)
}

func (s Server) updateTeamProblemSet(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	set, team, ok := s.teamProblemSetByParams(c)
	if !ok || !s.requireTeamManager(c, user, team) {
		return
	}
	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}
	if !bind(c, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "题单标题不能为空"})
		return
	}
	result := s.DB.Model(&models.TeamProblemSet{}).
		Where("id = ? AND team_id = ? AND deleted_at IS NULL", set.ID, team.ID).
		Updates(map[string]any{"title": req.Title, "description": strings.TrimSpace(req.Description)})
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected != 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem set not found"})
		return
	}
	_ = s.DB.Where("deleted_at IS NULL").First(&set, set.ID).Error
	services.Audit(c, s.DB, "team.problem_set.update", "team_problem_set", set.ID, datatypes.JSONMap{"team_id": team.ID})
	c.JSON(http.StatusOK, set)
}

func (s Server) deleteTeamProblemSet(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	set, team, ok := s.teamProblemSetByParams(c)
	if !ok || !s.requireTeamManager(c, user, team) {
		return
	}
	now := time.Now()
	result := s.DB.Model(&models.TeamProblemSet{}).
		Where("id = ? AND team_id = ? AND deleted_at IS NULL", set.ID, team.ID).
		Update("deleted_at", now)
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected != 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem set not found"})
		return
	}
	services.Audit(c, s.DB, "team.problem_set.delete", "team_problem_set", set.ID, datatypes.JSONMap{"team_id": team.ID})
	c.JSON(http.StatusOK, gin.H{"deleted": true, "problem_set_id": set.ID})
}

func (s Server) getTeamProblemSet(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	set, team, ok := s.teamProblemSetByParams(c)
	if !ok || !s.requireTeamMember(c, user, team) {
		return
	}
	var links []models.TeamProblemSetProblem
	s.DB.Preload("Problem").Where("problem_set_id = ?", set.ID).Order("sort_order, id").Find(&links)
	links = filterActiveTeamProblemLinks(links)
	normalizeTeamProblemLabels(links)
	canManage := s.canManageTeam(user, team)
	canOrganize := s.canTeamContentPermission(user, team)
	views := make([]teamProblemLinkView, 0, len(links))
	for _, link := range links {
		var latest models.Submission
		status := ""
		var submittedAt *time.Time
		var submissionCount int64
		var acceptedCount int64
		s.DB.Model(&models.Submission{}).Where("problem_set_id = ? AND problem_id = ?", set.ID, link.ProblemID).Count(&submissionCount)
		s.DB.Model(&models.Submission{}).Where("problem_set_id = ? AND problem_id = ? AND status = ?", set.ID, link.ProblemID, models.StatusAccepted).Count(&acceptedCount)
		if err := s.DB.Where("user_id = ? AND problem_set_id = ? AND problem_id = ?", user.ID, set.ID, link.ProblemID).Order("created_at desc").First(&latest).Error; err == nil {
			status = string(latest.Status)
			value := latest.CreatedAt
			submittedAt = &value
			var accepted int64
			s.DB.Model(&models.Submission{}).Where("user_id = ? AND problem_set_id = ? AND problem_id = ? AND status = ?", user.ID, set.ID, link.ProblemID, models.StatusAccepted).Count(&accepted)
			if accepted > 0 {
				status = string(models.StatusAccepted)
			}
		}
		views = append(views, teamProblemLinkView{
			TeamProblemSetProblem: link,
			SubmissionStatus:      status,
			SubmittedAt:           submittedAt,
			SubmissionCount:       submissionCount,
			AcceptedCount:         acceptedCount,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"problem_set": teamProblemSetView{
			TeamProblemSet: set,
			ProblemCount:   int64(len(views)),
			CanEdit:        canManage,
			CanDelete:      canManage,
		},
		"team":         s.teamViews([]models.Team{team}, user.ID)[0],
		"problems":     views,
		"can_organize": canOrganize,
		"can_edit":     canManage,
		"can_delete":   canManage,
	})
}

func (s Server) addTeamProblemSetProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	set, team, ok := s.teamProblemSetByParams(c)
	if !ok || !s.requireTeamContentPermission(c, user, team) {
		return
	}
	var req struct {
		ProblemID   uint   `json:"problem_id"`
		ProblemCode string `json:"problem_code"`
		Label       string `json:"label"`
		SortOrder   int    `json:"sort_order"`
	}
	if !bind(c, &req) {
		return
	}
	var problem models.Problem
	query := s.DB.Where("deleted_at IS NULL")
	if req.ProblemID > 0 {
		query = query.Where("id = ?", req.ProblemID)
	} else if code := strings.TrimSpace(req.ProblemCode); code != "" {
		query = query.Where("upper(display_code) = ?", strings.ToUpper(code))
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "problem_id or problem_code is required"})
		return
	}
	if err := query.First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return
	}
	if problem.TeamID != nil && *problem.TeamID != team.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能添加其他团队的私有题目"})
		return
	}
	if problem.TeamID == nil && !s.isProblemPublic(problem.ID) && !s.canManageProblemData(user, problem) {
		c.JSON(http.StatusForbidden, gin.H{"error": "题目尚未公开或无权使用"})
		return
	}
	link := models.TeamProblemSetProblem{ProblemSetID: set.ID, ProblemID: problem.ID, Label: strings.TrimSpace(req.Label), SortOrder: req.SortOrder}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := lockActiveTeamProblemSet(tx, set.ID); err != nil {
			return err
		}
		return tx.Create(&link).Error
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "problem set not found"})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "题目已在该题单中"})
		return
	}
	c.JSON(http.StatusCreated, link)
}

func (s Server) removeTeamProblemSetProblem(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	set, team, ok := s.teamProblemSetByParams(c)
	if !ok || !s.requireTeamContentPermission(c, user, team) {
		return
	}
	problemID, ok := idParam(c, "problem_id")
	if !ok {
		return
	}
	var removed bool
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := lockActiveTeamProblemSet(tx, set.ID); err != nil {
			return err
		}
		result := tx.Where("problem_set_id = ? AND problem_id = ?", set.ID, problemID).Delete(&models.TeamProblemSetProblem{})
		if result.Error != nil {
			return result.Error
		}
		removed = result.RowsAffected > 0
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "problem set not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"removed": removed})
}

func (s Server) listTeamProblemSetSubmissions(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	set, team, ok := s.teamProblemSetByParams(c)
	if !ok || !s.requireTeamMember(c, user, team) {
		return
	}
	var items []models.Submission
	s.DB.Where("problem_set_id = ?", set.ID).Order("id desc").Limit(500).Find(&items)
	views := s.submissionListViews(items)
	for index := range views {
		if views[index].UserID != user.ID {
			views[index].SourceCode = ""
		}
	}
	c.JSON(http.StatusOK, views)
}

func (s Server) createTeamProblemSetSubmission(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	set, team, ok := s.teamProblemSetByParams(c)
	if !ok || !s.requireTeamMember(c, user, team) {
		return
	}
	var req teamSubmissionInput
	if !bind(c, &req) {
		return
	}
	var count int64
	s.DB.Model(&models.TeamProblemSetProblem{}).Where("problem_set_id = ? AND problem_id = ?", set.ID, req.ProblemID).Count(&count)
	if count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "题目不属于当前题单"})
		return
	}
	s.createScopedTeamSubmission(c, user, req, nil, &set.ID)
}

func (s Server) listTeamDiscussions(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	set, team, ok := s.teamProblemSetByParams(c)
	if !ok || !s.requireTeamMember(c, user, team) {
		return
	}
	var rows []teamDiscussionView
	query := s.DB.Table("team_discussions").
		Select("team_discussions.*, users.name AS author_name, users.avatar_url AS author_avatar").
		Joins("JOIN users ON users.id = team_discussions.author_id").
		Where("team_discussions.problem_set_id = ?", set.ID).
		Order("team_discussions.created_at desc")
	if problemID := strings.TrimSpace(c.Query("problem_id")); problemID != "" {
		query = query.Where("team_discussions.problem_id = ?", problemID)
	}
	query.Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

func (s Server) createTeamDiscussion(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	set, team, ok := s.teamProblemSetByParams(c)
	if !ok || !s.requireTeamMember(c, user, team) {
		return
	}
	var req struct {
		ProblemID *uint  `json:"problem_id"`
		Content   string `json:"content" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if len([]rune(req.Content)) > 5000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "讨论内容不能超过 5000 个字符"})
		return
	}
	item := models.TeamDiscussion{ProblemSetID: set.ID, ProblemID: req.ProblemID, AuthorID: user.ID, Content: req.Content}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.TeamProblemSet
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).
			Where("id = ? AND deleted_at IS NULL", set.ID).
			First(&locked).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &teamContentMutationError{status: http.StatusNotFound, message: "problem set not found"}
			}
			return err
		}
		if req.ProblemID != nil {
			var count int64
			if err := tx.Model(&models.TeamProblemSetProblem{}).
				Where("problem_set_id = ? AND problem_id = ?", set.ID, *req.ProblemID).
				Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return &teamContentMutationError{status: http.StatusBadRequest, message: "题目不属于该题单"}
			}
		}
		return tx.Create(&item).Error
	}); err != nil {
		var mutationErr *teamContentMutationError
		if errors.As(err, &mutationErr) {
			c.JSON(mutationErr.status, gin.H{"error": mutationErr.message})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, teamDiscussionView{TeamDiscussion: item, AuthorName: user.Name, AuthorAvatar: user.AvatarURL})
}

func (s Server) teamByParam(c *gin.Context) (models.Team, bool) {
	value := strings.TrimSpace(c.Param("id"))
	var team models.Team
	query := s.DB
	if teamSlugPattern.MatchString(strings.ToLower(value)) {
		query = query.Where("slug = ?", strings.ToLower(value))
	} else {
		query = query.Where("id = ?", value)
	}
	if err := query.First(&team).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return models.Team{}, false
	}
	return team, true
}

func (s Server) teamByIDParam(c *gin.Context) (models.Team, bool) {
	id, ok := idParam(c, "id")
	if !ok {
		return models.Team{}, false
	}
	var team models.Team
	if err := s.DB.First(&team, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return models.Team{}, false
	}
	return team, true
}

func (s Server) teamProblemSetByParams(c *gin.Context) (models.TeamProblemSet, models.Team, bool) {
	setID, ok := idParam(c, "set_id")
	if !ok {
		return models.TeamProblemSet{}, models.Team{}, false
	}
	var set models.TeamProblemSet
	query := s.DB.Where("id = ? AND deleted_at IS NULL", setID)
	if rawTeamID := strings.TrimSpace(c.Param("id")); rawTeamID != "" {
		query = query.Where("team_id = ?", rawTeamID)
	}
	if err := query.First(&set).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem set not found"})
		return models.TeamProblemSet{}, models.Team{}, false
	}
	var team models.Team
	if err := s.DB.First(&team, set.TeamID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return models.TeamProblemSet{}, models.Team{}, false
	}
	return set, team, true
}

func (s Server) teamContestByParams(c *gin.Context) (models.TeamContest, models.Team, bool) {
	contestID, ok := idParam(c, "contest_id")
	if !ok {
		return models.TeamContest{}, models.Team{}, false
	}
	var contest models.TeamContest
	query := s.DB.Where("id = ? AND deleted_at IS NULL", contestID)
	if rawTeamID := strings.TrimSpace(c.Param("id")); rawTeamID != "" {
		query = query.Where("team_id = ?", rawTeamID)
	}
	if err := query.First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "contest not found"})
		return models.TeamContest{}, models.Team{}, false
	}
	s.syncTeamContestState(&contest, time.Now())
	var team models.Team
	if err := s.DB.First(&team, contest.TeamID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return models.TeamContest{}, models.Team{}, false
	}
	return contest, team, true
}

func (s Server) teamScopedProblem(c *gin.Context, user models.User, team models.Team, problemID uint, problemCode string) (models.Problem, bool) {
	var problem models.Problem
	query := s.DB.Where("deleted_at IS NULL")
	if problemID > 0 {
		query = query.Where("id = ?", problemID)
	} else if code := strings.TrimSpace(problemCode); code != "" {
		query = query.Where("upper(display_code) = ?", strings.ToUpper(code))
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "problem_id or problem_code is required"})
		return models.Problem{}, false
	}
	if err := query.First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem not found"})
		return models.Problem{}, false
	}
	if problem.TeamID != nil && *problem.TeamID != team.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能添加其他团队的私有题目"})
		return models.Problem{}, false
	}
	if problem.TeamID == nil && !s.isProblemPublic(problem.ID) && !s.canManageProblemData(user, problem) {
		c.JSON(http.StatusForbidden, gin.H{"error": "题目尚未公开或无权使用"})
		return models.Problem{}, false
	}
	return problem, true
}

func (s Server) createScopedTeamSubmission(c *gin.Context, user models.User, req teamSubmissionInput, contestID, problemSetID *uint) {
	if !validLanguage(req.Language) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "language must be one of c, cpp, python, java"})
		return
	}
	if len([]byte(req.SourceCode)) > services.MaxSubmissionSourceSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source code is too large"})
		return
	}
	submission := models.Submission{
		UserID:        user.ID,
		ProblemID:     req.ProblemID,
		TeamContestID: contestID,
		ProblemSetID:  problemSetID,
		Language:      req.Language,
		SourceCode:    req.SourceCode,
		IsPublic:      false,
		Status:        models.StatusQueued,
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if contestID != nil {
			var contest models.TeamContest
			if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).
				Where("id = ? AND deleted_at IS NULL", *contestID).
				First(&contest).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return &teamContentMutationError{status: http.StatusNotFound, message: "contest not found"}
				}
				return err
			}
			_, status := teamContestWindow(contest, time.Now())
			if status != models.TeamContestRunning {
				message := "比赛已结束"
				if status == models.TeamContestDraft || status == models.TeamContestPublished {
					message = "比赛尚未开始"
				}
				return &teamContentMutationError{status: http.StatusForbidden, message: message}
			}
			var participantCount int64
			if err := tx.Model(&models.TeamContestParticipant{}).
				Where("contest_id = ? AND user_id = ?", *contestID, user.ID).
				Count(&participantCount).Error; err != nil {
				return err
			}
			if participantCount == 0 {
				return &teamContentMutationError{status: http.StatusForbidden, message: "你不在比赛发布时冻结的参赛名单中"}
			}
			var linkCount int64
			if err := tx.Model(&models.TeamContestProblem{}).
				Where("contest_id = ? AND problem_id = ?", *contestID, req.ProblemID).
				Count(&linkCount).Error; err != nil {
				return err
			}
			if linkCount == 0 {
				return &teamContentMutationError{status: http.StatusBadRequest, message: "题目不属于当前比赛"}
			}
		}
		if problemSetID != nil {
			var set models.TeamProblemSet
			if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).
				Where("id = ? AND deleted_at IS NULL", *problemSetID).
				First(&set).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return &teamContentMutationError{status: http.StatusNotFound, message: "problem set not found"}
				}
				return err
			}
			var linkCount int64
			if err := tx.Model(&models.TeamProblemSetProblem{}).
				Where("problem_set_id = ? AND problem_id = ?", *problemSetID, req.ProblemID).
				Count(&linkCount).Error; err != nil {
				return err
			}
			if linkCount == 0 {
				return &teamContentMutationError{status: http.StatusBadRequest, message: "题目不属于当前题单"}
			}
		}
		var problem models.Problem
		if err := tx.Where("deleted_at IS NULL").First(&problem, req.ProblemID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &teamContentMutationError{status: http.StatusNotFound, message: "problem not found"}
			}
			return err
		}
		submission.ProblemID = problem.ID
		return tx.Create(&submission).Error
	}); err != nil {
		var mutationErr *teamContentMutationError
		if errors.As(err, &mutationErr) {
			c.JSON(mutationErr.status, gin.H{"error": mutationErr.message})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	streamID, err := streams.EnqueueSubmission(c.Request.Context(), s.Redis, submission.ID)
	if err != nil {
		s.DB.Model(&submission).Updates(map[string]any{"status": models.StatusSystemError, "message": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	action := "team.problem_set.submission.create"
	metadata := datatypes.JSONMap{"stream_id": streamID}
	if problemSetID != nil {
		metadata["problem_set_id"] = *problemSetID
	}
	if contestID != nil {
		action = "team.contest.submission.create"
		metadata = datatypes.JSONMap{"stream_id": streamID, "contest_id": *contestID}
	}
	services.Audit(c, s.DB, action, "submission", submission.ID, metadata)
	c.JSON(http.StatusCreated, submission)
}

func (s Server) teamMembership(teamID, userID uint) (models.TeamMembership, bool) {
	var membership models.TeamMembership
	if err := s.DB.Where("team_id = ? AND user_id = ?", teamID, userID).First(&membership).Error; err != nil {
		return models.TeamMembership{}, false
	}
	return membership, true
}

func (s Server) requireTeamMember(c *gin.Context, user models.User, team models.Team) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	if _, ok := s.teamMembership(team.ID, user.ID); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅团队成员可以访问"})
		return false
	}
	return true
}

func (s Server) requireTeamManager(c *gin.Context, user models.User, team models.Team) bool {
	if !s.canManageTeam(user, team) {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要团队管理员权限"})
		return false
	}
	return true
}

func (s Server) canManageTeam(user models.User, team models.Team) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	membership, ok := s.teamMembership(team.ID, user.ID)
	return hasTeamManagerPermission(user.Role, membership.Role, ok)
}

func hasTeamManagerPermission(userRole models.Role, membershipRole models.TeamRole, joined bool) bool {
	if userRole == models.RoleAdmin {
		return true
	}
	return joined && (membershipRole == models.TeamRoleOwner || membershipRole == models.TeamRoleAdmin)
}

func (s Server) requireTeamContentPermission(c *gin.Context, user models.User, team models.Team) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	membership, ok := s.teamMembership(team.ID, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅团队成员可以操作"})
		return false
	}
	allowed := membership.Role == models.TeamRoleOwner
	if team.ContestPermission == "admin" {
		allowed = allowed || membership.Role == models.TeamRoleAdmin
	}
	if team.ContestPermission == "all" {
		allowed = true
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "当前团队职级不能组织比赛或题单"})
	}
	return allowed
}

func (s Server) canTeamContentPermission(user models.User, team models.Team) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	membership, ok := s.teamMembership(team.ID, user.ID)
	if !ok {
		return false
	}
	if membership.Role == models.TeamRoleOwner {
		return true
	}
	if team.ContestPermission == "all" {
		return true
	}
	return team.ContestPermission == "admin" && membership.Role == models.TeamRoleAdmin
}

func (s Server) canAccessTeamProblem(user models.User, problem models.Problem) bool {
	if problem.TeamID == nil {
		return false
	}
	if user.Role == models.RoleAdmin {
		return true
	}
	_, ok := s.teamMembership(*problem.TeamID, user.ID)
	return ok
}

func (s Server) validateTeamProblemScope(c *gin.Context, user models.User, teamID, problemSetID *uint) bool {
	if teamID == nil {
		if problemSetID != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "problem_set_id requires team_id"})
			return false
		}
		return true
	}
	var team models.Team
	if err := s.DB.First(&team, *teamID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return false
	}
	if !s.requireTeamContentPermission(c, user, team) {
		return false
	}
	if problemSetID != nil {
		var count int64
		s.DB.Model(&models.TeamProblemSet{}).Where("id = ? AND team_id = ? AND deleted_at IS NULL", *problemSetID, team.ID).Count(&count)
		if count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "problem set not found"})
			return false
		}
	}
	return true
}

func (s Server) attachProblemToTeamScope(c *gin.Context, problemID uint, teamID, problemSetID *uint) bool {
	if teamID == nil {
		return true
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Problem{}).Where("id = ?", problemID).Update("team_id", *teamID).Error; err != nil {
			return err
		}
		if err := tx.Where("problem_id = ?", problemID).Delete(&models.ProblemReview{}).Error; err != nil {
			return err
		}
		if problemSetID != nil {
			var count int64
			if err := tx.Model(&models.TeamProblemSetProblem{}).Where("problem_set_id = ?", *problemSetID).Count(&count).Error; err != nil {
				return err
			}
			link := models.TeamProblemSetProblem{ProblemSetID: *problemSetID, ProblemID: problemID, SortOrder: int(count)}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func validTeamSetting(value string, allowed []string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func newTeamJoinCode() string {
	body := make([]byte, 6)
	if _, err := rand.Read(body); err != nil {
		return strings.ToUpper(hex.EncodeToString([]byte(time.Now().Format("150405"))))
	}
	return strings.ToUpper(hex.EncodeToString(body))
}

func normalizeTeamProblemLabels(links []models.TeamProblemSetProblem) {
	sort.SliceStable(links, func(i, j int) bool {
		if links[i].SortOrder == links[j].SortOrder {
			return links[i].ID < links[j].ID
		}
		return links[i].SortOrder < links[j].SortOrder
	})
	for index := range links {
		if strings.TrimSpace(links[index].Label) == "" {
			links[index].Label = fmt.Sprintf("P%04d", 1000+int(links[index].ID))
		}
	}
}

func normalizeTeamContestProblemLabels(links []models.TeamContestProblem) {
	sort.SliceStable(links, func(i, j int) bool {
		if links[i].SortOrder == links[j].SortOrder {
			return links[i].ID < links[j].ID
		}
		return links[i].SortOrder < links[j].SortOrder
	})
	for index := range links {
		if strings.TrimSpace(links[index].Label) == "" {
			links[index].Label = teamAlphaLabel(index)
		}
	}
}

func teamAlphaLabel(index int) string {
	index++
	label := ""
	for index > 0 {
		index--
		label = string(rune('A'+index%26)) + label
		index /= 26
	}
	return label
}

func resolveTeamContestAwardPercentages(gold, silver, bronze *int, fallbackGold, fallbackSilver, fallbackBronze int) (int, int, int, error) {
	goldPercent := fallbackGold
	if gold != nil {
		goldPercent = *gold
	}
	silverPercent := fallbackSilver
	if silver != nil {
		silverPercent = *silver
	}
	bronzePercent := fallbackBronze
	if bronze != nil {
		bronzePercent = *bronze
	}
	if goldPercent < 0 || goldPercent > 100 || silverPercent < 0 || silverPercent > 100 || bronzePercent < 0 || bronzePercent > 100 {
		return 0, 0, 0, errors.New("金、银、铜奖比例必须在 0 到 100 之间")
	}
	if goldPercent+silverPercent+bronzePercent > 100 {
		return 0, 0, 0, errors.New("金、银、铜奖比例之和不能超过 100")
	}
	return goldPercent, silverPercent, bronzePercent, nil
}

func teamContestPublishErrorResponse(err error) (int, string, bool) {
	switch {
	case errors.Is(err, errTeamContestDeleted):
		return http.StatusConflict, "比赛已删除，请刷新后重试", true
	case errors.Is(err, errTeamContestNoProblem):
		return http.StatusBadRequest, "发布前至少添加一道题目", true
	case errors.Is(err, gorm.ErrInvalidData):
		return http.StatusBadRequest, "发布前必须设置晚于当前时间的开始时间", true
	default:
		return 0, "", false
	}
}

func lockActiveTeamProblemSet(tx *gorm.DB, setID uint) error {
	var set models.TeamProblemSet
	return activeTeamProblemSetLockQuery(tx, setID).First(&set).Error
}

func activeTeamProblemSetLockQuery(tx *gorm.DB, setID uint) *gorm.DB {
	return tx.Clauses(clause.Locking{Strength: "SHARE"}).
		Where("id = ? AND deleted_at IS NULL", setID)
}

func teamContestAwardPercentValue(value *int) int {
	if value == nil {
		return defaultTeamContestAwardPercent
	}
	return *value
}

func teamContestWindow(contest models.TeamContest, now time.Time) (*time.Time, string) {
	if contest.State == "" {
		contest.State = models.TeamContestPublished
	}
	if contest.State == models.TeamContestDraft {
		return contestEndAt(contest), models.TeamContestDraft
	}
	if contest.State == models.TeamContestClosed {
		return contestEndAt(contest), models.TeamContestClosed
	}
	if contest.StartsAt == nil {
		return nil, models.TeamContestPublished
	}
	endsAt := contest.StartsAt.Add(time.Duration(contest.DurationMinutes) * time.Minute)
	if now.Before(*contest.StartsAt) {
		return &endsAt, models.TeamContestPublished
	}
	if !now.Before(endsAt) {
		return &endsAt, models.TeamContestClosed
	}
	return &endsAt, models.TeamContestRunning
}

func contestEndAt(contest models.TeamContest) *time.Time {
	if contest.StartsAt == nil {
		return nil
	}
	value := contest.StartsAt.Add(time.Duration(contest.DurationMinutes) * time.Minute)
	return &value
}

func requireDraftTeamContest(c *gin.Context, contest models.TeamContest) bool {
	if contest.State == "" || contest.State == models.TeamContestDraft {
		return true
	}
	c.JSON(http.StatusConflict, gin.H{"error": "比赛发布后题目和评分规则已冻结"})
	return false
}

func (s Server) syncTeamContestState(contest *models.TeamContest, now time.Time) {
	_, state := teamContestWindow(*contest, now)
	if contest.State == "" {
		contest.State = state
		return
	}
	if state == contest.State || contest.State == models.TeamContestDraft {
		return
	}
	if state == models.TeamContestRunning || state == models.TeamContestClosed {
		if result := s.DB.Model(contest).Where("state = ? AND deleted_at IS NULL", contest.State).Update("state", state); result.Error == nil && result.RowsAffected == 1 {
			contest.State = state
		}
	}
}

func filterActiveTeamProblemLinks(links []models.TeamProblemSetProblem) []models.TeamProblemSetProblem {
	active := make([]models.TeamProblemSetProblem, 0, len(links))
	for _, link := range links {
		if link.Problem.ID != 0 && link.Problem.DeletedAt == nil {
			active = append(active, link)
		}
	}
	return active
}

func filterActiveTeamContestProblemLinks(links []models.TeamContestProblem) []models.TeamContestProblem {
	active := make([]models.TeamContestProblem, 0, len(links))
	for _, link := range links {
		if link.Problem.ID != 0 && link.Problem.DeletedAt == nil {
			active = append(active, link)
		}
	}
	return active
}
