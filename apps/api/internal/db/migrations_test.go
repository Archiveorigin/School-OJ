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
	if len(files) < 22 {
		t.Fatalf("migration count = %d, want at least 22", len(files))
	}
	for index := 1; index < len(files); index++ {
		if files[index-1].Version >= files[index].Version {
			t.Fatalf("migrations not ordered at %q", files[index].Name)
		}
	}
	lifecycle := files[len(files)-2]
	if lifecycle.Version != 22 || lifecycle.Checksum == "" {
		t.Fatalf("lifecycle migration = %#v", lifecycle)
	}
	for _, column := range []string{"gold_award_percent", "silver_award_percent", "bronze_award_percent", "team_problem_sets", "deleted_at"} {
		if !strings.Contains(lifecycle.SQL, column) {
			t.Fatalf("lifecycle migration does not include %q", column)
		}
	}

	outbox := files[len(files)-1]
	if outbox.Version != 23 || outbox.Checksum == "" {
		t.Fatalf("outbox migration = %#v", outbox)
	}
	for _, column := range []string{"submission_id", "event_version", "payload", "available_at", "locked_until", "published_at"} {
		if !strings.Contains(outbox.SQL, column) {
			t.Fatalf("outbox migration does not include %q", column)
		}
	}
}
