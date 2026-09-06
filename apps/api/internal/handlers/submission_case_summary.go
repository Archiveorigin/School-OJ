package handlers

import (
	"gorm.io/datatypes"
	"school-oj/apps/api/internal/models"
)

type submissionCaseSummary struct {
	Passed int `json:"passed"`
	Total  int `json:"total"`
}

func summarizeSubmissionCases(status models.SubmissionStatus, manifest datatypes.JSONMap, results []models.SubmissionResult) *submissionCaseSummary {
	switch status {
	case models.StatusAccepted, models.StatusWrongAnswer, models.StatusCompileError, models.StatusRuntimeError, models.StatusTimeLimit, models.StatusMemoryLimit, models.StatusOutputLimit:
	default:
		return nil
	}
	tests := problemTestCases(manifest)
	if len(tests) == 0 {
		return nil
	}
	passed := map[string]bool{}
	for _, result := range results {
		if result.Status == models.StatusAccepted {
			passed[result.CaseName] = true
		}
	}
	summary := &submissionCaseSummary{Total: len(tests)}
	for _, test := range tests {
		if passed[test.Name] {
			summary.Passed++
		}
	}
	return summary
}
