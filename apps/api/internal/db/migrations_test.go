package db

import "testing"

func TestEmbeddedMigrationFilesAreOrderedAndIncludeLifecycle(t *testing.T) {
	files, err := embeddedMigrationFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 21 {
		t.Fatalf("migration count = %d, want at least 21", len(files))
	}
	for index := 1; index < len(files); index++ {
		if files[index-1].Version >= files[index].Version {
			t.Fatalf("migrations not ordered at %q", files[index].Name)
		}
	}
	last := files[len(files)-1]
	if last.Version != 21 || last.Checksum == "" {
		t.Fatalf("last migration = %#v", last)
	}
}
