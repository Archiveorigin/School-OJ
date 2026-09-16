package handlers

import "school-oj/apps/api/internal/models"

// Team submission access is scoped to the owning team, never to a base teacher
// role or the author of a public problem reused by an unrelated private team.
func (s Server) canManageTeamSubmission(user models.User, sub models.Submission) bool {
	var teamID, creatorID uint
	if sub.TeamContestID != nil {
		var contest models.TeamContest
		if s.DB.First(&contest, *sub.TeamContestID).Error != nil {
			return false
		}
		teamID, creatorID = contest.TeamID, contest.CreatedBy
	} else if sub.ProblemSetID != nil {
		var set models.TeamProblemSet
		if s.DB.First(&set, *sub.ProblemSetID).Error != nil {
			return false
		}
		teamID, creatorID = set.TeamID, set.CreatedBy
	} else {
		return false
	}
	var team models.Team
	if s.DB.First(&team, teamID).Error != nil {
		return false
	}
	return s.canManageTeam(user, team) || creatorID == user.ID && s.canTeamContentPermission(user, team)
}
