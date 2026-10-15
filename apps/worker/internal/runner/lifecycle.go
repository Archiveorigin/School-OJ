package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// These are the exact repo digests verified on the production judge host.
// Update deliberately and rerun the multi-language sandbox integration suite.
const (
	GCCImage    = "gcc@sha256:5e927c284bf55a7dc796262e311a0703344f62f41f5621eb56843111b1d37e15"
	PythonImage = "python@sha256:a39549e211a16149edf74e5fdc9ef03a6767e46cd987c5048b6659b6c9904c94"
	JavaImage   = "eclipse-temurin@sha256:949f7645022e8cd5cd65eac4b342e7505c86b721b6338f8b677c6f2edc180262"
)

func (r *DockerRunner) WarmImages(ctx context.Context) error {
	for _, ref := range []string{GCCImage, PythonImage, JavaImage} {
		if err := exec.CommandContext(ctx, "docker", "image", "inspect", ref).Run(); err == nil {
			continue
		}
		log.Printf("preparing sandbox image %s", ref)
		if out, err := exec.CommandContext(ctx, "docker", "pull", ref).CombinedOutput(); err != nil {
			return fmt.Errorf("prepare sandbox image %s: %w: %.1000s", ref, err, out)
		}
	}
	return nil
}

var sandboxDirectory = regexp.MustCompile(`^oj-[0-9]+-[a-zA-Z0-9]+$`)

// CleanupStale only touches this root's old sandboxes. Retained containers are
// intentional: OOM state must be inspected before removal. Never prune a daemon.
func (r *DockerRunner) CleanupStale(ctx context.Context) error {
	root, err := filepath.EvalSymlinks(r.Cfg.SandboxWorkRoot)
	if err != nil {
		return err
	}
	root, err = filepath.Abs(root)
	if err != nil || root == string(filepath.Separator) {
		return fmt.Errorf("unsafe sandbox cleanup root")
	}
	age := 2*r.Cfg.SubmissionTimeout + 10*time.Minute
	if age < time.Hour {
		age = time.Hour
	}
	cutoff := time.Now().Add(-age)
	out, err := exec.CommandContext(ctx, "docker", "ps", "-aq", "--filter", "name=school-oj-oj-").Output()
	if err != nil {
		return fmt.Errorf("list sandbox containers: %w", err)
	}
	active := map[string]bool{}
	for _, id := range strings.Fields(string(out)) {
		body, err := exec.CommandContext(ctx, "docker", "inspect", id).Output()
		if err != nil {
			return fmt.Errorf("inspect sandbox %s: %w", id, err)
		}
		var containers []struct {
			Created time.Time
			Mounts  []struct{ Source, Destination string }
		}
		if err := json.Unmarshal(body, &containers); err != nil || len(containers) != 1 {
			return fmt.Errorf("invalid sandbox inspection")
		}
		for _, mount := range containers[0].Mounts {
			if mount.Destination != "/work" || filepath.Dir(mount.Source) != root || !sandboxDirectory.MatchString(filepath.Base(mount.Source)) {
				continue
			}
			if containers[0].Created.Before(cutoff) {
				if err := exec.CommandContext(ctx, "docker", "rm", "-f", id).Run(); err != nil {
					return err
				}
				log.Printf("removed expired sandbox %s", id)
			} else {
				active[mount.Source] = true
			}
		}
	}
	return cleanupDirectories(root, cutoff, active)
}

func cleanupDirectories(root string, cutoff time.Time, active map[string]bool) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !sandboxDirectory.MatchString(entry.Name()) {
			continue
		}
		path := filepath.Join(root, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !active[path] && info.ModTime().Before(cutoff) {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
			log.Printf("removed expired sandbox directory %s", entry.Name())
		}
	}
	return nil
}
