package handlers

import "school-oj/apps/api/internal/models"

func applyProblemVersion(problem *models.Problem, version models.ProblemVersion) {
	if version.ID == 0 {
		return
	}
	problem.Title = version.Title
	problem.Statement = version.Statement
	problem.Tags = version.Tags
	problem.Difficulty = version.Difficulty
	problem.TimeLimitMS = version.TimeLimitMS
	problem.MemoryLimitMB = version.MemoryLimitMB
	problem.OutputLimitKB = version.OutputLimitKB
	problem.PackageObject = version.PackageObject
	problem.PackageChecksum = version.PackageChecksum
	problem.Manifest = version.Manifest
}

func hydrateExamProblemVersions(exam *models.Exam) {
	for index := range exam.Problems {
		applyProblemVersion(&exam.Problems[index].Problem, exam.Problems[index].ProblemVersion)
	}
}

func hydrateTeamContestProblemVersions(links []models.TeamContestProblem) {
	for index := range links {
		applyProblemVersion(&links[index].Problem, links[index].ProblemVersion)
	}
}
