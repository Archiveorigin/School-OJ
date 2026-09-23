package handlers

import (
	"encoding/json"
	"testing"

	"school-oj/apps/api/internal/models"
)

func TestSubmissionMineFilterPostgres(t *testing.T) {
	f := newExamFixture(t)
	create := func(user models.User, public bool, language string) models.Submission {
		sub := models.Submission{UserID: user.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Language: language, SourceCode: "print(3)", Status: models.StatusAccepted, IsPublic: public}
		if err := f.s.DB.Create(&sub).Error; err != nil {
			t.Fatal(err)
		}
		return sub
	}
	own := create(f.student, false, "python")
	create(f.admin, false, "python")
	create(f.peer, false, "java")
	// The own record must still be found when it is older than the normal 200-row window.
	for i := 0; i < 205; i++ {
		create(f.peer, true, "cpp")
	}
	list := func(path string, user models.User) []models.Submission {
		r := f.request("GET", path, &user, nil)
		assertStatus(t, r, 200)
		var rows []models.Submission
		if err := json.Unmarshal(r.Body.Bytes(), &rows); err != nil {
			t.Fatal(err)
		}
		return rows
	}
	for _, user := range []models.User{f.student, f.admin} {
		for _, path := range []string{"/api/submissions?mine=true", "/api/submissions?visibility=problem&problem_id=1&mine=true"} {
			rows := list(path, user)
			if len(rows) != 1 || rows[0].UserID != user.ID {
				t.Fatalf("mine filter returned unexpected rows: %+v", rows)
			}
		}
	}
	rows := list("/api/submissions?visibility=problem&problem_id=1&mine=true&language=python&username=考生甲", f.student)
	if len(rows) != 1 || rows[0].ID != own.ID || rows[0].IsPublic {
		t.Fatal("own private submission was lost")
	}
	if len(list("/api/submissions?visibility=problem&problem_id=1&mine=true&language=cpp", f.student)) != 0 {
		t.Fatal("language filter ignored")
	}
	rows = list("/api/submissions?visibility=problem&problem_id=1&mine=false", f.student)
	if len(rows) != 200 {
		t.Fatal("unchecking mine did not restore public records")
	}
	for _, row := range rows {
		if row.UserID != f.student.ID && !row.IsPublic {
			t.Fatal("private record leaked")
		}
	}
	assertStatus(t, f.request("GET", "/api/submissions?mine=invalid", &f.student, nil), 400)
}
