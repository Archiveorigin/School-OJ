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
	for _, f := range zr.File {
		name := filepath.ToSlash(filepath.Clean(f.Name))
		if strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") {
			return ProblemPackage{}, fmt.Errorf("unsafe zip path: %s", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return ProblemPackage{}, err
		}
		body, err := io.ReadAll(io.LimitReader(rc, 64<<20))
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
	for i := range manifest.Cases {
		if manifest.Cases[i].Name == "" {
			manifest.Cases[i].Name = fmt.Sprintf("case-%02d", i+1)
		}
		if manifest.Cases[i].Weight <= 0 {
			manifest.Cases[i].Weight = 100 / len(manifest.Cases)
		}
		if _, ok := files[filepath.ToSlash(filepath.Clean(manifest.Cases[i].Input))]; !ok {
			return ProblemPackage{}, fmt.Errorf("missing input for %s", manifest.Cases[i].Name)
		}
		if _, ok := files[filepath.ToSlash(filepath.Clean(manifest.Cases[i].Output))]; !ok {
			return ProblemPackage{}, fmt.Errorf("missing output for %s", manifest.Cases[i].Name)
		}
	}
	return ProblemPackage{Manifest: manifest, Files: files}, nil
}

func (p ProblemPackage) CaseInput(c Case) string {
	return string(p.Files[filepath.ToSlash(filepath.Clean(c.Input))])
}

func (p ProblemPackage) CaseOutput(c Case) string {
	return string(p.Files[filepath.ToSlash(filepath.Clean(c.Output))])
}
