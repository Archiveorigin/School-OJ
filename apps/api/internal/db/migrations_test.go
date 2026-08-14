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
	last := files[len(files)-1]
	if last.Version != 22 || last.Checksum == "" {
		t.Fatalf("last migration = %#v", last)
	}
	for _, column := range []string{"gold_award_percent", "silver_award_percent", "bronze_award_percent", "team_problem_sets", "deleted_at"} {
		if !strings.Contains(last.SQL, column) {
			t.Fatalf("last migration does not include %q", column)
		}
	}
}
