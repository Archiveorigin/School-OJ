package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
)

func (s Server) previewTeamJoin(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	team, ok := s.teamByParam(c)
	if !ok {
		return
	}
	view := s.teamViews([]models.Team{team}, user.ID)[0]
	c.JSON(200, gin.H{
		"id": team.ID, "name": team.Name, "slug": team.Slug, "icon_url": team.IconURL,
		"visibility": team.Visibility, "join_mode": team.JoinMode,
		"joined": view.Joined, "application_status": view.ApplicationState,
	})
}

// All join/review operations serialize on the team row before consulting state.
func (s Server) joinTeamTransaction(teamID, userID uint, code, message string) (bool, error) {
	joined := false
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var team models.Team
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&team, teamID).Error; err != nil {
			return err
		}
		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("account_deleted = false").First(&user, userID).Error; err != nil {
			return err
		}
		var membership models.TeamMembership
		err := tx.Where("team_id = ? AND user_id = ?", teamID, userID).First(&membership).Error
		if err == nil {
			joined = true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if team.JoinMode == "invitation" && (strings.TrimSpace(team.JoinCode) == "" || !strings.EqualFold(strings.TrimSpace(code), strings.TrimSpace(team.JoinCode))) {
			return &teamContentMutationError{403, "团队邀请码无效"}
		}
		if team.JoinMode == "application" {
			var application models.TeamJoinApplication
			err := tx.Where("team_id = ? AND user_id = ? AND status = 'pending'", teamID, userID).First(&application).Error
			if err == nil {
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			return tx.Create(&models.TeamJoinApplication{TeamID: teamID, UserID: userID, Status: "pending", Message: strings.TrimSpace(message)}).Error
		}
		if team.JoinMode != "open" && team.JoinMode != "invitation" {
			return &teamContentMutationError{409, "团队准入方式无效"}
		}
		membership = models.TeamMembership{TeamID: teamID, UserID: userID, Role: models.TeamRoleMember}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&membership).Error; err != nil {
			return err
		}
		joined = true
		return tx.Model(&models.TeamJoinApplication{}).Where("team_id = ? AND user_id = ? AND status = 'pending'", teamID, userID).Updates(map[string]any{"status": "approved", "reviewed_at": time.Now()}).Error
	})
	return joined, err
}

func (s Server) reviewTeamApplicationTransaction(actor models.User, teamID, applicationID uint, action string) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var team models.Team
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&team, teamID).Error; err != nil {
			return err
		}
		scoped := s
		scoped.DB = tx
		if !scoped.canManageTeam(actor, team) {
			return &teamContentMutationError{403, "需要团队管理员权限"}
		}
		var application models.TeamJoinApplication
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND team_id = ?", applicationID, teamID).First(&application).Error; err != nil {
			return err
		}
		status := "rejected"
		if action == "approve" {
			status = "approved"
		}
		if application.Status == status {
			return nil
		}
		if application.Status != "pending" {
			return &teamContentMutationError{409, "申请已经处理，请刷新"}
		}
		if action == "approve" {
			var user models.User
			if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("account_deleted = false").First(&user, application.UserID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return &teamContentMutationError{409, "申请人账号已删除"}
				}
				return err
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.TeamMembership{TeamID: teamID, UserID: application.UserID, Role: models.TeamRoleMember}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&application).Updates(map[string]any{"status": status, "reviewed_by": actor.ID, "reviewed_at": time.Now()}).Error
	})
}

// Membership changes share the join/review lock, so a stale approval cannot
// reinstate a removed member and revoked managers cannot race a new decision.
func (s Server) changeTeamMembership(actor models.User, teamID, targetID uint, action string, role models.TeamRole) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var team models.Team
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&team, teamID).Error; err != nil {
			return err
		}
		scoped := s
		scoped.DB = tx
		if action != "leave" && !scoped.canManageTeam(actor, team) {
			return &teamContentMutationError{403, "需要团队管理员权限"}
		}
		var target models.TeamMembership
		if err := tx.Where("team_id = ? AND user_id = ?", teamID, targetID).First(&target).Error; err != nil {
			return err
		}
		manager, _ := scoped.teamMembership(teamID, actor.ID)
		owner := actor.Role == models.RoleAdmin || manager.Role == models.TeamRoleOwner
		switch action {
		case "leave":
			if actor.ID != targetID || target.Role == models.TeamRoleOwner {
				return &teamContentMutationError{400, "创建者不能退出团队，请先转让团队"}
			}
		case "remove":
			if target.Role == models.TeamRoleOwner || (target.Role == models.TeamRoleAdmin && !owner) {
				return &teamContentMutationError{403, "没有权限移除该成员"}
			}
		case "role":
			if !owner || target.Role == models.TeamRoleOwner {
				return &teamContentMutationError{403, "只有创建者可以调整成员职级"}
			}
			if role != models.TeamRoleMember && role != models.TeamRoleAdmin {
				return &teamContentMutationError{400, "成员职级无效"}
			}
			return tx.Model(&target).Update("role", role).Error
		default:
			return &teamContentMutationError{400, "成员操作无效"}
		}
		return tx.Delete(&target).Error
	})
}
