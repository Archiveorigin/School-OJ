package services

import (
	"context"
	"school-oj/apps/api/internal/domain/team"
	"school-oj/apps/api/internal/models"
	"strings"
)

type TeamAccess struct{ Store team.Repository }

func (s TeamAccess) PreviewInvitation(ctx context.Context, code string, userID uint) (team.Preview, error) {
	code = team.NormalizeCode(code)
	if len(code) < 6 || len(code) > 24 {
		return team.Preview{}, team.Fail(team.InvalidInvitation, "邀请码无效或已失效，请向团队管理员确认")
	}
	value, err := s.Store.Invitation(ctx, code)
	if err != nil {
		return team.Preview{}, err
	}
	return s.Store.Preview(ctx, value, userID)
}
func (s TeamAccess) Join(ctx context.Context, teamID, userID uint, code, message string) (bool, error) {
	message = strings.TrimSpace(message)
	if len([]rune(message)) > 300 || len(strings.TrimSpace(code)) > 24 {
		return false, team.Fail(team.Invalid, "邀请码或申请说明过长")
	}
	joined := false
	err := s.Store.WithTeam(ctx, teamID, func(tx team.Transaction) error {
		if _, err := tx.ActiveUser(userID); err != nil {
			return err
		}
		_, member, err := tx.Membership(userID)
		if err != nil {
			return err
		}
		if member {
			joined = true
			return tx.CompletePending(userID)
		}
		value := tx.Team()
		switch value.JoinMode {
		case "invitation":
			if team.NormalizeCode(value.JoinCode) == "" || team.NormalizeCode(value.JoinCode) != team.NormalizeCode(code) {
				return team.Fail(team.InvalidInvitation, "团队邀请码无效或已失效")
			}
		case "application":
			_, pending, err := tx.PendingApplication(userID)
			if err != nil || pending {
				return err
			}
			return tx.CreateApplication(userID, message)
		case "open":
		default:
			return team.Fail(team.Conflict, "团队准入方式无效")
		}
		if err := tx.AddMember(userID); err != nil {
			return err
		}
		if err := tx.CompletePending(userID); err != nil {
			return err
		}
		joined = true
		return nil
	})
	return joined && err == nil, err
}
func (s TeamAccess) Review(ctx context.Context, actorID, teamID, applicationID uint, action string) error {
	return s.Store.WithTeam(ctx, teamID, func(tx team.Transaction) error {
		actor, err := tx.ActiveUser(actorID)
		if err != nil {
			return err
		}
		membership, joined, err := tx.Membership(actorID)
		if err != nil {
			return err
		}
		if !team.CanManage(actor.Role, membership.Role, joined) {
			return team.Fail(team.Forbidden, "需要团队管理员权限")
		}
		application, err := tx.Application(applicationID)
		if err != nil {
			return err
		}
		status, replay, err := team.ReviewTransition(application.Status, action)
		if err != nil || replay {
			return err
		}
		if action == "approve" {
			if _, err := tx.ActiveUser(application.UserID); err != nil {
				return err
			}
			if err := tx.AddMember(application.UserID); err != nil {
				return err
			}
		}
		return tx.Review(application, status, actorID)
	})
}
func (s TeamAccess) ChangeMembership(ctx context.Context, actorID, teamID, targetID uint, action string, role models.TeamRole) error {
	return s.Store.WithTeam(ctx, teamID, func(tx team.Transaction) error {
		actor, err := tx.ActiveUser(actorID)
		if err != nil {
			return err
		}
		manager, joined, err := tx.Membership(actorID)
		if err != nil {
			return err
		}
		if action != "leave" && !team.CanManage(actor.Role, manager.Role, joined) {
			return team.Fail(team.Forbidden, "需要团队管理员权限")
		}
		target, exists, err := tx.Membership(targetID)
		if err != nil {
			return err
		}
		if !exists {
			return team.Fail(team.NotFound, "团队成员不存在")
		}
		owner := team.CanEditPermissions(actor.Role, manager.Role, joined)
		switch action {
		case "leave":
			if actorID != targetID || target.Role == models.TeamRoleOwner {
				return team.Fail(team.Invalid, "创建者不能退出团队")
			}
		case "remove":
			if target.Role == models.TeamRoleOwner || (target.Role == models.TeamRoleAdmin && !owner) {
				return team.Fail(team.Forbidden, "没有权限移除该成员")
			}
		case "role":
			if !owner || target.Role == models.TeamRoleOwner {
				return team.Fail(team.Forbidden, "只有创建者可以调整成员职级")
			}
			if role != models.TeamRoleMember && role != models.TeamRoleAdmin {
				return team.Fail(team.Invalid, "成员职级只能设为管理员或团员")
			}
			return tx.SetRole(target, role)
		default:
			return team.Fail(team.Invalid, "成员操作无效")
		}
		return tx.Remove(target)
	})
}
