package services

import (
	"fmt"
	"strings"
)

type ProblemStatementSample struct {
	Input  string `json:"input" yaml:"input"`
	Output string `json:"output" yaml:"output"`
}
type ProblemStatementSections struct {
	Description  string                   `json:"description" yaml:"description"`
	InputFormat  string                   `json:"input_format" yaml:"input_format"`
	OutputFormat string                   `json:"output_format" yaml:"output_format"`
	Samples      []ProblemStatementSample `json:"samples" yaml:"samples"`
	Notes        string                   `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// The canonical Markdown and structured source travel together in the pinned
// problem package, so editing and previewing use exactly the same content.
func RenderProblemStatement(s *ProblemStatementSections) (string, error) {
	if s == nil {
		return "", fmt.Errorf("请填写题目描述、输入格式、输出格式和样例")
	}
	if strings.TrimSpace(s.Description) == "" || strings.TrimSpace(s.InputFormat) == "" || strings.TrimSpace(s.OutputFormat) == "" || len(s.Samples) < 1 || len(s.Samples) > 20 {
		return "", fmt.Errorf("题目描述、输入格式、输出格式必填，样例需为 1 至 20 组")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## 题目描述\n\n%s\n\n## 输入格式\n\n%s\n\n## 输出格式\n\n%s\n", strings.TrimSpace(s.Description), strings.TrimSpace(s.InputFormat), strings.TrimSpace(s.OutputFormat))
	for i, sample := range s.Samples {
		if strings.TrimSpace(sample.Output) == "" {
			return "", fmt.Errorf("第 %d 组样例输出不能为空", i+1)
		}
		for _, part := range []struct{ label, value string }{{"输入", sample.Input}, {"输出", sample.Output}} {
			if len(part.value) > 65536 {
				return "", fmt.Errorf("样例过长，请将大型数据放入测试点文件")
			}
			fence := "```"
			for strings.Contains(part.value, fence) {
				fence += "`"
			}
			fmt.Fprintf(&b, "\n## 样例 %d · %s\n\n%stext\n%s\n%s\n", i+1, part.label, fence, strings.TrimSuffix(part.value, "\n"), fence)
		}
	}
	if strings.TrimSpace(s.Notes) != "" {
		fmt.Fprintf(&b, "\n## 说明\n\n%s\n", strings.TrimSpace(s.Notes))
	}
	if b.Len() > 200000 {
		return "", fmt.Errorf("完整题面不能超过 200000 字节")
	}
	return b.String(), nil
}
