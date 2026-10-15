package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupDirectoriesPreservesActiveRecentAndUnrelatedData(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-2 * time.Hour)
	for _, name := range []string{"oj-1-stale", "oj-2-active", "oj-3-recent", "unrelated", "seccomp"} {
		path := filepath.Join(root, name)
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
		if name != "oj-3-recent" {
			if err := os.Chtimes(path, old, old); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := cleanupDirectories(root, time.Now().Add(-time.Hour), map[string]bool{filepath.Join(root, "oj-2-active"): true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "oj-1-stale")); !os.IsNotExist(err) {
		t.Fatal("stale source retained")
	}
	for _, name := range []string{"oj-2-active", "oj-3-recent", "unrelated", "seccomp"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("removed %s", name)
		}
	}
}

func TestLanguageImagesArePinned(t *testing.T) {
	for _, lang := range []string{"c", "cpp", "python", "java"} {
		spec, err := languageSpec(lang)
		if err != nil || len(spec.Image) < 72 {
			t.Fatalf("unpinned image for %s", lang)
		}
	}
}
