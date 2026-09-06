package db

import (
	"strings"
	"testing"
)

func TestEmbeddedMigrationFilesAreOrderedAndIncludeLifecycle(t *testing.T) {
	files, err := embeddedMigrationFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 25 {
		t.Fatalf("migration count = %d, want at least 25", len(files))
	}
	for index := 1; index < len(files); index++ {
		if files[index-1].Version >= files[index].Version {
			t.Fatalf("migrations not ordered at %q", files[index].Name)
		}
	}
	byVersion := map[int]migrationFile{}
	for _, file := range files {
		byVersion[file.Version] = file
	}
	lifecycle := byVersion[22]
	if lifecycle.Version != 22 || lifecycle.Checksum == "" {
		t.Fatalf("lifecycle migration = %#v", lifecycle)
	}
	for _, column := range []string{"gold_award_percent", "silver_award_percent", "bronze_award_percent", "team_problem_sets", "deleted_at"} {
		if !strings.Contains(lifecycle.SQL, column) {
			t.Fatalf("lifecycle migration does not include %q", column)
		}
	}

	outbox := byVersion[23]
	if outbox.Version != 23 || outbox.Checksum == "" {
		t.Fatalf("outbox migration = %#v", outbox)
	}
	for _, column := range []string{"submission_id", "event_version", "payload", "available_at", "locked_until", "published_at"} {
		if !strings.Contains(outbox.SQL, column) {
			t.Fatalf("outbox migration does not include %q", column)
		}
	}

	changeTickets := byVersion[24]
	if changeTickets.Version != 24 || changeTickets.Checksum == "" {
		t.Fatalf("problem change migration = %#v", changeTickets)
	}
	for _, required := range []string{"problem_versions", "problem_change_tickets", "problem_version_id", "archived_at", "freeze_enabled", "freeze_duration_minutes", "'oi'", "'ioi'", "'acm'"} {
		if !strings.Contains(changeTickets.SQL, required) {
			t.Fatalf("problem change migration does not include %q", required)
		}
	}
	activity := byVersion[25]
	for _, required := range []string{"exam_activities", "event_id", "client_at", "ip", "meta", "UNIQUE"} {
		if activity.Checksum == "" || !strings.Contains(activity.SQL, required) {
			t.Fatalf("activity migration does not include %q", required)
		}
	}
}
