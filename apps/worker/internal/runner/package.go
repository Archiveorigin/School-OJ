package runner

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const utf8BOM = "\ufeff"

const maxProblemPackageSize = 128 << 20

type ProblemPackage struct {
	Manifest Manifest
	Files    map[string][]byte
}

type Manifest struct {
	Slug          string `yaml:"slug"`
	Title         string `yaml:"title"`
	Statement     string `yaml:"statement"`
	TimeLimitMS   int    `yaml:"time_limit_ms"`
	MemoryLimitMB int    `yaml:"memory_limit_mb"`
	OutputLimitKB int    `yaml:"output_limit_kb"`
	Cases         []Case `yaml:"cases"`
}

type Case struct {
	Name   string `yaml:"name"`
	Input  string `yaml:"input"`
	Output string `yaml:"output"`
	Weight int    `yaml:"weight"`
}

func ParsePackage(body []byte) (ProblemPackage, error) {
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return ProblemPackage{}, err
	}
	files := map[string][]byte{}
	totalSize := int64(0)
	for _, f := range zr.File {
		rawName := filepath.ToSlash(f.Name)
		if unsafeZipPath(rawName) {
			return ProblemPackage{}, fmt.Errorf("unsafe zip path: %s", f.Name)
		}
		name := filepath.ToSlash(filepath.Clean(rawName))
		if f.FileInfo().IsDir() {
			continue
		}
		if err := validatePackageFilePath(name); err != nil {
			return ProblemPackage{}, err
		}
		totalSize += int64(f.UncompressedSize64)
		if totalSize > maxProblemPackageSize {
			return ProblemPackage{}, fmt.Errorf("problem package is too large")
		}
		rc, err := f.Open()
		if err != nil {
			return ProblemPackage{}, err
		}
		body, err := readLimited(rc, maxProblemPackageSize, "problem package")
		_ = rc.Close()
		if err != nil {
			return ProblemPackage{}, err
		}
		files[name] = body
	}
	raw, ok := files["problem.yaml"]
	if !ok {
		return ProblemPackage{}, fmt.Errorf("problem.yaml not found")
	}
	var manifest Manifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return ProblemPackage{}, err
	}
	if manifest.TimeLimitMS <= 0 {
		manifest.TimeLimitMS = 1000
	}
	if manifest.MemoryLimitMB <= 0 {
		manifest.MemoryLimitMB = 256
	}
	if manifest.OutputLimitKB <= 0 {
		manifest.OutputLimitKB = 1024
	}
	if len(manifest.Cases) == 0 {
		return ProblemPackage{}, fmt.Errorf("at least one test case is required")
	}
	for i := range manifest.Cases {
		if manifest.Cases[i].Name == "" {
			manifest.Cases[i].Name = fmt.Sprintf("case-%02d", i+1)
		}
		if manifest.Cases[i].Weight <= 0 {
			manifest.Cases[i].Weight = 100 / len(manifest.Cases)
		}
		input, err := normalizePackageTestPath(manifest.Cases[i].Input, ".in")
		if err != nil {
			return ProblemPackage{}, fmt.Errorf("case %s: %w", manifest.Cases[i].Name, err)
		}
		output, err := normalizePackageTestPath(manifest.Cases[i].Output, ".out")
		if err != nil {
			return ProblemPackage{}, fmt.Errorf("case %s: %w", manifest.Cases[i].Name, err)
		}
		if _, ok := files[input]; !ok {
			return ProblemPackage{}, fmt.Errorf("missing input for %s", manifest.Cases[i].Name)
		}
		if _, ok := files[output]; !ok {
			return ProblemPackage{}, fmt.Errorf("missing output for %s", manifest.Cases[i].Name)
		}
		manifest.Cases[i].Input = input
		manifest.Cases[i].Output = output
	}
	return ProblemPackage{Manifest: manifest, Files: files}, nil
}

func (p ProblemPackage) CaseInput(c Case) string {
	return stripUTF8BOM(string(p.Files[filepath.ToSlash(filepath.Clean(c.Input))]))
}

func (p ProblemPackage) CaseOutput(c Case) string {
	return stripUTF8BOM(string(p.Files[filepath.ToSlash(filepath.Clean(c.Output))]))
}

func stripUTF8BOM(value string) string {
	return strings.TrimPrefix(value, utf8BOM)
}

func readLimited(r io.Reader, maxBytes int64, label string) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("%s is too large", label)
	}
	return body, nil
}

func validatePackageFilePath(clean string) error {
	if clean == "problem.yaml" {
		return nil
	}
	if strings.HasPrefix(clean, "tests/") {
		_, err := normalizePackageTestPath(clean, "")
		return err
	}
	if strings.HasPrefix(clean, "assets/") {
		ext := strings.ToLower(filepath.Ext(clean))
		switch ext {
		case ".png", ".jpg", ".jpeg", ".gif", ".webp":
			return nil
		default:
			return fmt.Errorf("asset type is not supported: %s", clean)
		}
	}
	return fmt.Errorf("unsupported problem package file: %s", clean)
}

func unsafeZipPath(value string) bool {
	if strings.HasPrefix(value, "/") || strings.Contains(value, "\x00") {
		return true
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func normalizePackageTestPath(value string, requiredExt string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
	if clean == "." || clean == "" || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || strings.Contains(clean, "\x00") {
		return "", fmt.Errorf("unsafe test path: %s", value)
	}
	if !strings.HasPrefix(clean, "tests/") || strings.TrimPrefix(clean, "tests/") == "" {
		return "", fmt.Errorf("test path must be under tests/: %s", value)
	}
	ext := strings.ToLower(filepath.Ext(clean))
	if ext != ".in" && ext != ".out" {
		return "", fmt.Errorf("test file type is not supported: %s", value)
	}
	if requiredExt != "" && ext != requiredExt {
		return "", fmt.Errorf("test file must use %s: %s", requiredExt, value)
	}
	return clean, nil
}
