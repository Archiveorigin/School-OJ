package handlers

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"school-oj/apps/api/internal/services"
)

func TestRouterBuilds(t *testing.T) {
	_ = (Server{}).Router()
}

func TestPreparedProblemInputDraftKeepsAssets(t *testing.T) {
	req := preparedProblemInput{
		Slug:          "image-prepared-problem",
		Title:         "Image prepared problem",
		Statement:     "![diagram](assets/diagram.png)",
		TimeLimitMS:   1000,
		MemoryLimitMB: 256,
		OutputLimitKB: 1024,
		Assets: []services.ProblemAssetDraft{
			{
				Name:        "diagram.png",
				Path:        "assets/diagram.png",
				ContentType: "image/png",
				Data:        "data:image/png;base64,iVBORw0KGgo=",
			},
		},
		Cases: []services.ProblemCaseDraft{
			{Name: "case-01", Input: "1 2\n", Output: "3\n", Weight: 100},
		},
	}

	draft := req.draft()
	if len(draft.Assets) != 1 {
		t.Fatalf("expected one asset, got %d", len(draft.Assets))
	}
	if draft.Assets[0].Path != "assets/diagram.png" {
		t.Fatalf("unexpected asset path: %s", draft.Assets[0].Path)
	}
	if draft.Assets[0].Data == "" {
		t.Fatal("expected asset data to be preserved")
	}
}

func TestBuildXLSXIncludesExamReportRows(t *testing.T) {
	body, err := buildXLSX([][]xlsxCell{
		{xlsxString("学生姓名"), xlsxString("学号"), xlsxString("通过题目数"), xlsxString("所得分数")},
		{xlsxString("张三"), xlsxString("20260001"), xlsxNumber(2), xlsxNumber(180)},
	})
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	var sheet string
	for _, file := range zr.File {
		if file.Name != "xl/worksheets/sheet1.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		sheet = string(raw)
	}
	if !strings.Contains(sheet, "张三") || !strings.Contains(sheet, "<v>180</v>") {
		t.Fatalf("worksheet does not contain report row: %s", sheet)
	}
}
