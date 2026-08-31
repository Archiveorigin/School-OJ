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
	if len(files) < 24 {
		t.Fatalf("migration count = %d, want at least 24", len(files))
	}
	for index := 1; index < len(files); index++ {
		if files[index-1].Version >= files[index].Version {
			t.Fatalf("migrations not ordered at %q", files[index].Name)
		}
	}
	lifecycle := files[len(files)-3]
	if lifecycle.Version != 22 || lifecycle.Checksum == "" {
		t.Fatalf("lifecycle migration = %#v", lifecycle)
	}
	for _, column := range []string{"gold_award_percent", "silver_award_percent", "bronze_award_percent", "team_problem_sets", "deleted_at"} {
		if !strings.Contains(lifecycle.SQL, column) {
			t.Fatalf("lifecycle migration does not include %q", column)
		}
	}

	outbox := files[len(files)-2]
	if outbox.Version != 23 || outbox.Checksum == "" {
		t.Fatalf("outbox migration = %#v", outbox)
	}
	for _, column := range []string{"submission_id", "event_version", "payload", "available_at", "locked_until", "published_at"} {
		if !strings.Contains(outbox.SQL, column) {
			t.Fatalf("outbox migration does not include %q", column)
		}
	}

	changeTickets := files[len(files)-1]
	if changeTickets.Version != 24 || changeTickets.Checksum == "" {
		t.Fatalf("problem change migration = %#v", changeTickets)
	}
	for _, required := range []string{"problem_versions", "problem_change_tickets", "problem_version_id", "archived_at", "freeze_enabled", "freeze_duration_minutes", "'oi'", "'ioi'", "'acm'"} {
		if !strings.Contains(changeTickets.SQL, required) {
			t.Fatalf("problem change migration does not include %q", required)
		}
	}
}
