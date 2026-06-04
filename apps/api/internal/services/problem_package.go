package services

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProblemManifest struct {
	Slug          string          `json:"slug" yaml:"slug"`
	Title         string          `json:"title" yaml:"title"`
	Statement     string          `json:"statement" yaml:"statement"`
	TimeLimitMS   int             `json:"time_limit_ms" yaml:"time_limit_ms"`
	MemoryLimitMB int             `json:"memory_limit_mb" yaml:"memory_limit_mb"`
	OutputLimitKB int             `json:"output_limit_kb" yaml:"output_limit_kb"`
	Assets        []AssetManifest `json:"assets,omitempty" yaml:"assets,omitempty"`
	Cases         []CaseManifest  `json:"cases" yaml:"cases"`
}

type AssetManifest struct {
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Path        string `json:"path" yaml:"path"`
	ContentType string `json:"content_type,omitempty" yaml:"content_type,omitempty"`
	Size        int64  `json:"size,omitempty" yaml:"size,omitempty"`
	Object      string `json:"object,omitempty" yaml:"-"`
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
	Assets   []ParsedProblemAsset
}

type ParsedProblemAsset struct {
	Name        string
	Path        string
	ContentType string
	Size        int64
	Body        []byte
}

type ProblemPackageDraft struct {
	Slug          string              `json:"slug"`
	Title         string              `json:"title"`
	Statement     string              `json:"statement"`
	Tags          []string            `json:"tags"`
	TimeLimitMS   int                 `json:"time_limit_ms"`
	MemoryLimitMB int                 `json:"memory_limit_mb"`
	OutputLimitKB int                 `json:"output_limit_kb"`
	ClassIDs      []uint              `json:"class_ids"`
	Assets        []ProblemAssetDraft `json:"assets"`
	Cases         []ProblemCaseDraft  `json:"cases"`
}

type ProblemAssetDraft struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	Data        string `json:"data"`
}

type ProblemCaseDraft struct {
	Name   string `json:"name"`
	Input  string `json:"input"`
	Output string `json:"output"`
	Weight int    `json:"weight"`
}

type TestPointUploadFile struct {
	Name string
	Body []byte
}

const (
	MaxProblemAssetSize     = 5 << 20
	MaxProblemAssetsSize    = 20 << 20
	MaxProblemTestFilesSize = 128 << 20
	utf8BOM                 = "\ufeff"
)

type rawTestPointFile struct {
	Name string
	Body []byte
}

type testPointGroup struct {
	inputName  string
	outputName string
	input      []byte
	output     []byte
}

func BuildProblemCasesFromTestPointFiles(files []TestPointUploadFile) ([]ProblemCaseDraft, error) {
	expanded, err := expandTestPointUploadFiles(files)
	if err != nil {
		return nil, err
	}
	groups := map[int]*testPointGroup{}
	for _, file := range expanded {
		clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(file.Name)))
		ext := strings.ToLower(filepath.Ext(clean))
		if ext != ".in" && ext != ".out" {
			return nil, fmt.Errorf("test file %s is not a .in or .out file", file.Name)
		}
		stem := strings.TrimSuffix(filepath.Base(clean), filepath.Ext(clean))
		seq, ok := lastNumber(stem)
		if !ok {
			return nil, fmt.Errorf("test file %s must contain a numeric case id", file.Name)
		}
		group := groups[seq]
		if group == nil {
			group = &testPointGroup{}
			groups[seq] = group
		}
		if ext == ".in" {
			if group.inputName != "" {
				return nil, fmt.Errorf("duplicate input file for case %d", seq)
			}
			group.inputName = clean
			group.input = file.Body
			continue
		}
		if group.outputName != "" {
			return nil, fmt.Errorf("duplicate output file for case %d", seq)
		}
		group.outputName = clean
		group.output = file.Body
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("at least one test case is required")
	}
	seqs := make([]int, 0, len(groups))
	for seq := range groups {
		seqs = append(seqs, seq)
	}
	sort.Ints(seqs)
	cases := make([]ProblemCaseDraft, 0, len(seqs))
	for _, seq := range seqs {
		group := groups[seq]
		if group.inputName == "" || group.outputName == "" {
			missing := ".in"
			if group.inputName != "" {
				missing = ".out"
			}
			return nil, fmt.Errorf("case %d is missing %s file", seq, missing)
		}
		cases = append(cases, ProblemCaseDraft{
			Name:   fmt.Sprintf("case-%02d", len(cases)+1),
			Input:  normalizeCaseText(string(group.input)),
			Output: normalizeCaseText(string(group.output)),
			Weight: 1,
		})
	}
	return cases, nil
}

func expandTestPointUploadFiles(files []TestPointUploadFile) ([]rawTestPointFile, error) {
	var expanded []rawTestPointFile
	totalSize := 0
	for _, file := range files {
		name := strings.TrimSpace(file.Name)
		if name == "" {
			return nil, fmt.Errorf("test file name is required")
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".zip" {
			totalSize += len(file.Body)
			if totalSize > MaxProblemTestFilesSize {
				return nil, fmt.Errorf("test files are too large")
			}
			expanded = append(expanded, rawTestPointFile{Name: name, Body: file.Body})
			continue
		}
		if len(file.Body) > MaxProblemTestFilesSize {
			return nil, fmt.Errorf("test zip %s is too large", name)
		}
		reader, err := zip.NewReader(bytes.NewReader(file.Body), int64(len(file.Body)))
		if err != nil {
			return nil, fmt.Errorf("open test zip %s: %w", name, err)
		}
		for _, item := range reader.File {
			rawPath := filepath.ToSlash(item.Name)
			if unsafeZipPath(rawPath) {
				return nil, fmt.Errorf("unsafe zip path: %s", item.Name)
			}
			if item.FileInfo().IsDir() {
				continue
			}
			clean := filepath.ToSlash(filepath.Clean(rawPath))
			ext := strings.ToLower(filepath.Ext(clean))
			if ext != ".in" && ext != ".out" {
				return nil, fmt.Errorf("test zip %s contains unsupported file %s", name, item.Name)
			}
			rc, err := item.Open()
			if err != nil {
				return nil, err
			}
			body, err := io.ReadAll(io.LimitReader(rc, MaxProblemTestFilesSize+1))
			_ = rc.Close()
			if err != nil {
				return nil, err
			}
			totalSize += len(body)
			if len(body) > MaxProblemTestFilesSize || totalSize > MaxProblemTestFilesSize {
				return nil, fmt.Errorf("test files are too large")
			}
			expanded = append(expanded, rawTestPointFile{Name: clean, Body: body})
		}
	}
	return expanded, nil
}

func lastNumber(value string) (int, bool) {
	end := -1
	start := -1
	for i := len(value) - 1; i >= 0; i-- {
		if value[i] < '0' || value[i] > '9' {
			continue
		}
		end = i + 1
		start = i
		for i > 0 && value[i-1] >= '0' && value[i-1] <= '9' {
			i--
			start = i
		}
		break
	}
	if start < 0 || end <= start {
		return 0, false
	}
	seq, err := strconv.Atoi(value[start:end])
	if err != nil || seq <= 0 {
		return 0, false
	}
	return seq, true
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
	files := map[string][]byte{}
	for i, tc := range draft.Cases {
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
		files[inputPath] = []byte(normalizeCaseText(tc.Input))
		files[outputPath] = []byte(normalizeCaseText(tc.Output))
	}
	totalAssetSize := int64(0)
	usedAssetPaths := map[string]bool{}
	for i, asset := range draft.Assets {
		path, err := NormalizeAssetPath(asset.Path)
		if err != nil {
			return nil, ParsedProblemPackage{}, fmt.Errorf("asset %d: %w", i+1, err)
		}
		if usedAssetPaths[path] {
			return nil, ParsedProblemPackage{}, fmt.Errorf("duplicate asset path: %s", path)
		}
		body, err := DecodeProblemAssetData(asset.Data)
		if err != nil {
			return nil, ParsedProblemPackage{}, fmt.Errorf("asset %s: %w", path, err)
		}
		if len(body) == 0 {
			return nil, ParsedProblemPackage{}, fmt.Errorf("asset %s is empty", path)
		}
		if len(body) > MaxProblemAssetSize {
			return nil, ParsedProblemPackage{}, fmt.Errorf("asset %s is too large", path)
		}
		totalAssetSize += int64(len(body))
		if totalAssetSize > MaxProblemAssetsSize {
			return nil, ParsedProblemPackage{}, fmt.Errorf("problem assets are too large")
		}
		contentType, err := ValidateProblemAsset(path, body)
		if err != nil {
			return nil, ParsedProblemPackage{}, err
		}
		name := strings.TrimSpace(asset.Name)
		if name == "" {
			name = filepath.Base(path)
		}
		manifest.Assets = append(manifest.Assets, AssetManifest{
			Name:        name,
			Path:        path,
			ContentType: contentType,
			Size:        int64(len(body)),
		})
		files[path] = body
		usedAssetPaths[path] = true
	}
	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, ParsedProblemPackage{}, err
	}
	files["problem.yaml"] = manifestBytes

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, ParsedProblemPackage{}, err
		}
		if _, err := w.Write(body); err != nil {
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

func RebuildProblemPackage(base []byte, manifest ProblemManifest, replacementCases []ProblemCaseDraft) ([]byte, ParsedProblemPackage, error) {
	files, err := packageFiles(base)
	if err != nil {
		return nil, ParsedProblemPackage{}, err
	}
	delete(files, "problem.yaml")
	if replacementCases != nil {
		if len(replacementCases) == 0 {
			return nil, ParsedProblemPackage{}, fmt.Errorf("at least one test case is required")
		}
		for name := range files {
			if strings.HasPrefix(name, "tests/") {
				delete(files, name)
			}
		}
		manifest.Cases = make([]CaseManifest, 0, len(replacementCases))
		for i, tc := range replacementCases {
			name := strings.TrimSpace(tc.Name)
			if name == "" {
				name = fmt.Sprintf("case-%02d", i+1)
			}
			weight := tc.Weight
			if weight <= 0 {
				weight = 100 / len(replacementCases)
			}
			inputPath := fmt.Sprintf("tests/%02d.in", i+1)
			outputPath := fmt.Sprintf("tests/%02d.out", i+1)
			manifest.Cases = append(manifest.Cases, CaseManifest{
				Name:   name,
				Input:  inputPath,
				Output: outputPath,
				Weight: weight,
			})
			files[inputPath] = []byte(normalizeCaseText(tc.Input))
			files[outputPath] = []byte(normalizeCaseText(tc.Output))
		}
	}
	if strings.TrimSpace(manifest.Slug) == "" || strings.TrimSpace(manifest.Title) == "" {
		return nil, ParsedProblemPackage{}, fmt.Errorf("slug and title are required")
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
	manifest.Slug = strings.TrimSpace(manifest.Slug)
	manifest.Title = strings.TrimSpace(manifest.Title)
	manifest.Statement = strings.TrimSpace(manifest.Statement)
	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, ParsedProblemPackage{}, err
	}
	files["problem.yaml"] = manifestBytes
	body, err := buildProblemZip(files)
	if err != nil {
		return nil, ParsedProblemPackage{}, err
	}
	parsed, err := ParseProblemPackage(body)
	if err != nil {
		return nil, ParsedProblemPackage{}, err
	}
	return body, parsed, nil
}

func packageFiles(body []byte) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	files := map[string][]byte{}
	for _, item := range reader.File {
		rawPath := filepath.ToSlash(item.Name)
		if unsafeZipPath(rawPath) {
			return nil, fmt.Errorf("unsafe zip path: %s", item.Name)
		}
		if item.FileInfo().IsDir() {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(rawPath))
		rc, err := item.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(rc, 128<<20))
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		files[clean] = data
	}
	return files, nil
}

func buildProblemZip(files map[string][]byte) ([]byte, error) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if _, err := w.Write(files[name]); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ParseProblemPackage(body []byte) (ParsedProblemPackage, error) {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return ParsedProblemPackage{}, fmt.Errorf("open zip: %w", err)
	}
	files := map[string]bool{}
	assets := map[string]ParsedProblemAsset{}
	totalAssetSize := int64(0)
	var manifestBytes []byte
	for _, f := range reader.File {
		rawPath := filepath.ToSlash(f.Name)
		if unsafeZipPath(rawPath) {
			return ParsedProblemPackage{}, fmt.Errorf("unsafe zip path: %s", f.Name)
		}
		clean := filepath.ToSlash(filepath.Clean(rawPath))
		if f.FileInfo().IsDir() {
			continue
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
			continue
		}
		if strings.HasPrefix(clean, "assets/") {
			path, err := NormalizeAssetPath(clean)
			if err != nil {
				return ParsedProblemPackage{}, err
			}
			if f.UncompressedSize64 > MaxProblemAssetSize {
				return ParsedProblemPackage{}, fmt.Errorf("asset %s is too large", path)
			}
			totalAssetSize += int64(f.UncompressedSize64)
			if totalAssetSize > MaxProblemAssetsSize {
				return ParsedProblemPackage{}, fmt.Errorf("problem assets are too large")
			}
			rc, err := f.Open()
			if err != nil {
				return ParsedProblemPackage{}, err
			}
			assetBody, err := io.ReadAll(io.LimitReader(rc, MaxProblemAssetSize+1))
			_ = rc.Close()
			if err != nil {
				return ParsedProblemPackage{}, err
			}
			if len(assetBody) > MaxProblemAssetSize {
				return ParsedProblemPackage{}, fmt.Errorf("asset %s is too large", path)
			}
			contentType, err := ValidateProblemAsset(path, assetBody)
			if err != nil {
				return ParsedProblemPackage{}, err
			}
			assets[path] = ParsedProblemAsset{
				Name:        filepath.Base(path),
				Path:        path,
				ContentType: contentType,
				Size:        int64(len(assetBody)),
				Body:        assetBody,
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
	for i := range manifest.Assets {
		asset := &manifest.Assets[i]
		path, err := NormalizeAssetPath(asset.Path)
		if err != nil {
			return ParsedProblemPackage{}, err
		}
		parsedAsset, ok := assets[path]
		if !ok {
			return ParsedProblemPackage{}, fmt.Errorf("asset references missing file %s", asset.Path)
		}
		asset.Path = path
		if strings.TrimSpace(asset.Name) == "" {
			asset.Name = parsedAsset.Name
		}
		asset.ContentType = parsedAsset.ContentType
		asset.Size = parsedAsset.Size
	}
	if len(manifest.Assets) == 0 && len(assets) > 0 {
		for _, asset := range assets {
			manifest.Assets = append(manifest.Assets, AssetManifest{
				Name:        asset.Name,
				Path:        asset.Path,
				ContentType: asset.ContentType,
				Size:        asset.Size,
			})
		}
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
		Assets:   sortedAssets(assets),
	}, nil
}

func normalizeCaseText(value string) string {
	value = strings.TrimPrefix(value, utf8BOM)
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	if value != "" && !strings.HasSuffix(value, "\n") {
		value += "\n"
	}
	return value
}

func NormalizeAssetPath(value string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
	if clean == "." || clean == "" || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("unsafe asset path: %s", value)
	}
	if !strings.HasPrefix(clean, "assets/") || strings.TrimPrefix(clean, "assets/") == "" {
		return "", fmt.Errorf("asset path must be under assets/: %s", value)
	}
	if strings.Contains(clean, "\x00") {
		return "", fmt.Errorf("unsafe asset path: %s", value)
	}
	if _, ok := assetContentTypeForExt(clean); !ok {
		return "", fmt.Errorf("asset type is not supported: %s", value)
	}
	return clean, nil
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

func DecodeProblemAssetData(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("asset data is required")
	}
	if comma := strings.Index(value, ","); strings.HasPrefix(value, "data:") && comma >= 0 {
		value = value[comma+1:]
	}
	body, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode asset data: %w", err)
	}
	return body, nil
}

func ValidateProblemAsset(path string, body []byte) (string, error) {
	contentType, ok := assetContentTypeForExt(path)
	if !ok {
		return "", fmt.Errorf("asset type is not supported: %s", path)
	}
	detected := http.DetectContentType(body)
	if detected == contentType || (contentType == "image/jpeg" && detected == "image/jpg") {
		return contentType, nil
	}
	if contentType == "image/webp" && (detected == "application/octet-stream" || detected == "image/webp") {
		return contentType, nil
	}
	return "", fmt.Errorf("asset %s content does not match %s", path, contentType)
}

func assetContentTypeForExt(path string) (string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png", true
	case ".jpg", ".jpeg":
		return "image/jpeg", true
	case ".gif":
		return "image/gif", true
	case ".webp":
		return "image/webp", true
	default:
		return "", false
	}
}

func sortedAssets(items map[string]ParsedProblemAsset) []ParsedProblemAsset {
	out := make([]ParsedProblemAsset, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Path < out[i].Path {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
