package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"school-oj/apps/api/internal/models"
)

// nil/null means all eligible team members. An empty list deliberately ranks
// nobody. This list never controls access to the contest or submission APIs.
func normalizeContestRankingRoster(db *gorm.DB, teamID uint, raw []byte) (datatypes.JSON, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return datatypes.JSON("null"), nil
	}
	var ids []uint
	if err := json.Unmarshal(raw, &ids); err != nil || len(ids) > 100000 {
		return nil, fmt.Errorf("排名名单必须为团队成员 ID 数组")
	}
	seen := make(map[uint]bool, len(ids))
	unique := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			return nil, fmt.Errorf("排名名单包含无效用户")
		}
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	var count int64
	if len(unique) > 0 {
		if err := db.Model(&models.TeamMembership{}).Joins("JOIN users ON users.id = team_memberships.user_id").Where("team_memberships.team_id = ? AND users.account_deleted = false AND users.id IN ?", teamID, unique).Count(&count).Error; err != nil {
			return nil, err
		}
		if count != int64(len(unique)) {
			return nil, fmt.Errorf("排名名单包含非团队成员或已注销用户，请重新选择")
		}
	}
	sort.Slice(unique, func(i, j int) bool { return unique[i] < unique[j] })
	encoded, err := json.Marshal(unique)
	return datatypes.JSON(encoded), err
}

func contestRankingIDs(raw datatypes.JSON) ([]uint, bool, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, false, nil
	}
	var ids []uint
	err := json.Unmarshal(raw, &ids)
	return ids, true, err
}
