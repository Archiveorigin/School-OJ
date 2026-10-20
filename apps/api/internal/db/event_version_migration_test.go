package db

import (
	"school-oj/apps/api/internal/models"
	"testing"
)

func TestEventSubmissionVersionMigrationPostgres(t *testing.T) {
	database := migrationTestDB(t)
	if err := Migrate(database); err != nil {
		t.Fatal(err)
	}
	// Populate legacy audit records and rerun only the additive backfill. Separate
	// event kinds can share an ID, deleted events still retain their audit history.
	for _, entry := range []models.EventReviewLog{
		{EventKind: "exam", EventID: 17, Revision: 8, Action: "submit", Note: "", Snapshot: []byte(`{}`)},
		{EventKind: "contest", EventID: 17, Revision: 3, Action: "submit", Note: "", Snapshot: []byte(`{}`)},
		{EventKind: "exam", EventID: 17, Revision: 9, Action: "reject", Note: "", Snapshot: []byte(`{}`)},
		{EventKind: "exam", EventID: 17, Revision: 21, Action: "submit", Note: "", Snapshot: []byte(`{}`)},
		{EventKind: "exam", EventID: 17, Revision: 22, Action: "delete", Note: "", Snapshot: []byte(`{}`)},
	} {
		if err := database.Create(&entry).Error; err != nil {
			t.Fatal(err)
		}
	}
	files, err := embeddedMigrationFiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Version == 34 {
			for i := 0; i < 2; i++ {
				if err := database.Exec(f.SQL).Error; err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	var entries []models.EventReviewLog
	database.Order("id").Find(&entries)
	want := []int{1, 1, 1, 2, 2}
	for i, entry := range entries {
		if entry.SubmissionVersion != want[i] {
			t.Fatalf("entry %d version %d", i, entry.SubmissionVersion)
		}
	}
}
