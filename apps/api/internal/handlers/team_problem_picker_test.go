package handlers

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	"school-oj/apps/api/internal/models"
)

func TestTeamProblemPickerPostgres(t *testing.T) {
	f := newExamFixture(t)
	team := models.Team{Name: "批量选题", Slug: "batch-picker", OwnerID: f.teacher.ID, Visibility: "private", JoinMode: "open", ContestPermission: "owner"}
	if err := f.s.DB.Create(&team).Error; err != nil {
		t.Fatal(err)
	}
	f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.teacher.ID, Role: models.TeamRoleOwner})
	f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.student.ID, Role: models.TeamRoleMember})
	set := models.TeamProblemSet{TeamID: team.ID, Title: "批量题单", CreatedBy: f.teacher.ID}
	f.s.DB.Create(&set)
	private := models.Problem{Title: "团队私有题", Slug: "own-team-picker", DisplayCode: "PICK1", TeamID: &team.ID, Difficulty: "基础"}
	if err := f.s.DB.Create(&private).Error; err != nil {
		t.Fatal(err)
	}
	otherTeam := models.Team{Name: "其他团队", Slug: "foreign-team-picker", OwnerID: f.peer.ID, Visibility: "private", JoinMode: "open", ContestPermission: "owner"}
	if err := f.s.DB.Create(&otherTeam).Error; err != nil {
		t.Fatal(err)
	}
	otherID := otherTeam.ID
	foreign := models.Problem{Title: "其他团队题", Slug: "foreign-picker", DisplayCode: "PICK2", TeamID: &otherID}
	if err := f.s.DB.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/api/problem-sets/%d/problems", set.ID)
	assertStatus(t, f.request("POST", path, &f.student, gin.H{"problem_ids": []uint{f.problem.ID}}), 403)
	assertStatus(t, f.request("POST", path, &f.teacher, gin.H{"problem_ids": []uint{f.problem.ID, foreign.ID}}), 403)
	var count int64
	f.s.DB.Model(&models.TeamProblemSetProblem{}).Where("problem_set_id = ?", set.ID).Count(&count)
	if count != 0 {
		t.Fatal("partial batch persisted")
	}
	for i := 0; i < 2; i++ {
		r := f.request("POST", path, &f.teacher, gin.H{"problem_ids": []uint{f.problem.ID, private.ID, f.problem.ID}})
		assertStatus(t, r, 201)
		var result struct{ Added, Skipped int }
		_ = json.Unmarshal(r.Body.Bytes(), &result)
		if i == 0 && result.Added != 2 || i == 1 && result.Skipped != 2 {
			t.Fatalf("bad idempotent batch: %+v", result)
		}
	}
	assertStatus(t, f.request("POST", path, &f.teacher, gin.H{"problem_ids": []uint{}}), 400)
	assertStatus(t, f.request("POST", path, &f.teacher, gin.H{"problem_ids": make([]uint, 101)}), 400)
	catalog := fmt.Sprintf("/api/problems/catalog?team_id=%d&q=PICK&difficulty=基础", team.ID)
	assertStatus(t, f.request("GET", catalog, &f.student, nil), 403)
	r := f.request("GET", catalog, &f.teacher, nil)
	assertStatus(t, r, 200)
	var page struct{ Items []models.Problem }
	_ = json.Unmarshal(r.Body.Bytes(), &page)
	if len(page.Items) != 1 || page.Items[0].ID != private.ID {
		t.Fatalf("private picker scope failed: %+v", page.Items)
	}
	editPath := fmt.Sprintf("/api/problem-sets/%d", set.ID)
	assertStatus(t, f.request("PUT", editPath, &f.teacher, gin.H{"title": set.Title, "icon_url": "/logo1.png"}), 200)
	assertStatus(t, f.request("PUT", editPath, &f.teacher, gin.H{"title": "保留图标"}), 200)
	f.s.DB.First(&set, set.ID)
	if set.IconURL != "/logo1.png" {
		t.Fatal("omitted icon cleared existing value")
	}
	assertStatus(t, f.request("PUT", editPath, &f.teacher, gin.H{"title": set.Title, "icon_url": "javascript:bad"}), 400)
}
