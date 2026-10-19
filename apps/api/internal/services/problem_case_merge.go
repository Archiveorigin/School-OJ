package services

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
)

// Unlike the legacy importer, this preserves the uploaded pair name. Renumbering
// a partial upload would replace case 1 when the author uploaded only case 9.
func BuildNamedProblemCasesFromTestPointFiles(files []TestPointUploadFile) ([]ProblemCaseDraft, error) {
	expanded, err := expandTestPointUploadFiles(files)
	if err != nil {
		return nil, err
	}
	type pair struct{ input, output *rawTestPointFile }
	pairs := map[string]*pair{}
	for i := range expanded {
		file := &expanded[i]
		name := strings.ReplaceAll(strings.TrimSpace(file.Name), "\\", "/")
		if unsafeZipPath(name) {
			return nil, fmt.Errorf("测试文件路径无效：%s", name)
		}
		ext := strings.ToLower(path.Ext(name))
		if ext != ".in" && ext != ".out" {
			return nil, fmt.Errorf("测试文件须为 .in 或 .out")
		}
		key := strings.ToLower(strings.TrimPrefix(strings.TrimSuffix(path.Clean(name), path.Ext(name)), "tests/"))
		if key == "" {
			return nil, fmt.Errorf("测试点名称不能为空")
		}
		if pairs[key] == nil {
			pairs[key] = &pair{}
		}
		p := pairs[key]
		if ext == ".in" {
			if p.input != nil {
				return nil, fmt.Errorf("输入文件重复：%s", name)
			}
			p.input = file
		} else {
			if p.output != nil {
				return nil, fmt.Errorf("输出文件重复：%s", name)
			}
			p.output = file
		}
	}
	if len(pairs) == 0 || len(pairs) > MaxProblemCases {
		return nil, fmt.Errorf("测试点须为 1 至 %d 组", MaxProblemCases)
	}
	keys := make([]string, 0, len(pairs))
	for key := range pairs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	cases := make([]ProblemCaseDraft, 0, len(keys))
	for _, key := range keys {
		p := pairs[key]
		if p.input == nil || p.output == nil {
			return nil, fmt.Errorf("测试点 %s 缺少配对的 .in 或 .out", key)
		}
		cases = append(cases, ProblemCaseDraft{Name: key, Input: string(p.input.Body), Output: string(p.output.Body), Weight: 1})
	}
	return cases, nil
}

// MergeProblemCases updates only named pairs. Existing order and weights survive;
// omitted cases are never deleted. Full replacement is a separate explicit mode.
func MergeProblemCases(base []byte, replacements []ProblemCaseDraft) ([]ProblemCaseDraft, error) {
	pkg, err := ParseProblemPackage(base)
	if err != nil {
		return nil, err
	}
	z, err := zip.NewReader(bytes.NewReader(base), int64(len(base)))
	if err != nil {
		return nil, err
	}
	files := map[string]*zip.File{}
	for _, file := range z.File {
		files[file.Name] = file
	}
	read := func(name string) (string, error) {
		f, ok := files[name]
		if !ok {
			return "", fmt.Errorf("missing test file %s", name)
		}
		r, e := f.Open()
		if e != nil {
			return "", e
		}
		defer r.Close()
		b, e := io.ReadAll(io.LimitReader(r, MaxProblemTestFilesSize+1))
		if len(b) > MaxProblemTestFilesSize {
			return "", fmt.Errorf("test file too large")
		}
		return string(b), e
	}
	canonical := func(name string) string { return strings.ToLower(strings.TrimPrefix(path.Clean(name), "tests/")) }
	indices := map[string]int{}
	out := make([]ProblemCaseDraft, 0, len(pkg.Manifest.Cases)+len(replacements))
	for _, tc := range pkg.Manifest.Cases {
		input, e := read(tc.Input)
		if e != nil {
			return nil, e
		}
		output, e := read(tc.Output)
		if e != nil {
			return nil, e
		}
		index := len(out)
		out = append(out, ProblemCaseDraft{Name: tc.Name, Input: input, Output: output, Weight: tc.Weight})
		for _, key := range []string{canonical(tc.Name), canonical(strings.TrimSuffix(tc.Input, path.Ext(tc.Input)))} {
			if prior, exists := indices[key]; exists && prior != index {
				return nil, fmt.Errorf("测试点名称不唯一，不能安全合并：%s", tc.Name)
			}
			indices[key] = index
		}
	}
	seen := map[int]bool{}
	for _, tc := range replacements {
		if index, ok := indices[canonical(tc.Name)]; ok {
			if seen[index] {
				return nil, fmt.Errorf("同一测试点被重复指定：%s", tc.Name)
			}
			seen[index] = true
			out[index].Input = tc.Input
			out[index].Output = tc.Output
		} else {
			index := len(out)
			indices[canonical(tc.Name)] = index
			seen[index] = true
			out = append(out, tc)
		}
	}
	if len(out) > MaxProblemCases {
		return nil, fmt.Errorf("合并后测试点不能超过 %d 个", MaxProblemCases)
	}
	return out, nil
}
