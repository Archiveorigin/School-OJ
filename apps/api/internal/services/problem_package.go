package services

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProblemManifest struct {
	Slug          string         `json:"slug" yaml:"slug"`
	Title         string         `json:"title" yaml:"title"`
	Statement     string         `json:"statement" yaml:"statement"`
	TimeLimitMS   int            `json:"time_limit_ms" yaml:"time_limit_ms"`
	MemoryLimitMB int            `json:"memory_limit_mb" yaml:"memory_limit_mb"`
	OutputLimitKB int            `json:"output_limit_kb" yaml:"output_limit_kb"`
	Cases         []CaseManifest `json:"cases" yaml:"cases"`
}

type CaseManifest struct {
	Name   string `json:"name" yaml:"name"`
	Input  string `json:"input" yaml:"input"`
	Output string `json:"output" yaml:"output"`
	Weight int    `json:"weight" yaml:"weight"`
}

type ParsedProblemPackage struct {
	Manifest ProblemManifest
	SHA256   string
	Size     int64
}

type ProblemPackageDraft struct {
	Slug          string             `json:"slug"`
	Title         string             `json:"title"`
	Statement     string             `json:"statement"`
	TimeLimitMS   int                `json:"time_limit_ms"`
	MemoryLimitMB int                `json:"memory_limit_mb"`
	OutputLimitKB int                `json:"output_limit_kb"`
	Cases         []ProblemCaseDraft `json:"cases"`
}

type ProblemCaseDraft struct {
	Name   string `json:"name"`
	Input  string `json:"input"`
	Output string `json:"output"`
	Weight int    `json:"weight"`
}

func BuildProblemPackage(draft ProblemPackageDraft) ([]byte, ParsedProblemPackage, error) {
	if strings.TrimSpace(draft.Slug) == "" || strings.TrimSpace(draft.Title) == "" {
		return nil, ParsedProblemPackage{}, fmt.Errorf("slug and title are required")
	}
	if len(draft.Cases) == 0 {
		return nil, ParsedProblemPackage{}, fmt.Errorf("at least one test case is required")
	}
	if draft.TimeLimitMS <= 0 {
		draft.TimeLimitMS = 1000
	}
	if draft.MemoryLimitMB <= 0 {
		draft.MemoryLimitMB = 256
	}
	if draft.OutputLimitKB <= 0 {
		draft.OutputLimitKB = 1024
	}

	manifest := ProblemManifest{
		Slug:          strings.TrimSpace(draft.Slug),
		Title:         strings.TrimSpace(draft.Title),
		Statement:     strings.TrimSpace(draft.Statement),
		TimeLimitMS:   draft.TimeLimitMS,
		MemoryLimitMB: draft.MemoryLimitMB,
		OutputLimitKB: draft.OutputLimitKB,
		Cases:         make([]CaseManifest, 0, len(draft.Cases)),
	}
	files := map[string]string{}
	for i, tc := range draft.Cases {
		if strings.TrimSpace(tc.Input) == "" {
			return nil, ParsedProblemPackage{}, fmt.Errorf("case %d input is required", i+1)
		}
		if strings.TrimSpace(tc.Output) == "" {
			return nil, ParsedProblemPackage{}, fmt.Errorf("case %d output is required", i+1)
		}
		name := strings.TrimSpace(tc.Name)
		if name == "" {
			name = fmt.Sprintf("case-%02d", i+1)
		}
		weight := tc.Weight
		if weight <= 0 {
			weight = 100 / len(draft.Cases)
		}
		inputPath := fmt.Sprintf("tests/%02d.in", i+1)
		outputPath := fmt.Sprintf("tests/%02d.out", i+1)
		manifest.Cases = append(manifest.Cases, CaseManifest{
			Name:   name,
			Input:  inputPath,
			Output: outputPath,
			Weight: weight,
		})
		files[inputPath] = normalizeCaseText(tc.Input)
		files[outputPath] = normalizeCaseText(tc.Output)
	}
	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, ParsedProblemPackage{}, err
	}
	files["problem.yaml"] = string(manifestBytes)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, ParsedProblemPackage{}, err
		}
		if _, err := io.WriteString(w, body); err != nil {
			return nil, ParsedProblemPackage{}, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, ParsedProblemPackage{}, err
	}
	body := buf.Bytes()
	parsed, err := ParseProblemPackage(body)
	if err != nil {
		return nil, ParsedProblemPackage{}, err
	}
	return body, parsed, nil
}

func ParseProblemPackage(body []byte) (ParsedProblemPackage, error) {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return ParsedProblemPackage{}, fmt.Errorf("open zip: %w", err)
	}
	files := map[string]bool{}
	var manifestBytes []byte
	for _, f := range reader.File {
		clean := filepath.ToSlash(filepath.Clean(f.Name))
		if strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
			return ParsedProblemPackage{}, fmt.Errorf("unsafe zip path: %s", f.Name)
		}
		files[clean] = true
		if clean == "problem.yaml" {
			rc, err := f.Open()
			if err != nil {
				return ParsedProblemPackage{}, err
			}
			manifestBytes, err = io.ReadAll(io.LimitReader(rc, 1<<20))
			_ = rc.Close()
			if err != nil {
				return ParsedProblemPackage{}, err
			}
		}
	}
	if len(manifestBytes) == 0 {
		return ParsedProblemPackage{}, fmt.Errorf("problem.yaml is required")
	}
	var manifest ProblemManifest
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		return ParsedProblemPackage{}, fmt.Errorf("parse problem.yaml: %w", err)
	}
	if manifest.Slug == "" || manifest.Title == "" {
		return ParsedProblemPackage{}, fmt.Errorf("slug and title are required")
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
		return ParsedProblemPackage{}, fmt.Errorf("at least one test case is required")
	}
	for i := range manifest.Cases {
		tc := &manifest.Cases[i]
		if tc.Name == "" {
			tc.Name = fmt.Sprintf("case-%02d", i+1)
		}
		if tc.Weight <= 0 {
			tc.Weight = 100 / len(manifest.Cases)
		}
		for _, path := range []string{tc.Input, tc.Output} {
			clean := filepath.ToSlash(filepath.Clean(path))
			if clean == "." || strings.HasPrefix(clean, "../") || !files[clean] {
				return ParsedProblemPackage{}, fmt.Errorf("case %s references missing file %s", tc.Name, path)
			}
		}
	}
	sum := sha256.Sum256(body)
	return ParsedProblemPackage{
		Manifest: manifest,
		SHA256:   hex.EncodeToString(sum[:]),
		Size:     int64(len(body)),
	}, nil
}

func normalizeCaseText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	if !strings.HasSuffix(value, "\n") {
		value += "\n"
	}
	return value
}
