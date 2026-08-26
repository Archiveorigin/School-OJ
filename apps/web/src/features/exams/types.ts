import type {
  ClassContext,
  PreparedProblem,
  Problem,
  ScoringRule,
} from "../../api/client";

export type CourseSummary = {
  id: number;
  code: string;
  name: string;
};

export type ExamDraft = {
  course_id?: number;
  class_id?: number;
  title: string;
  description: string;
  starts_at: Date | string | null;
  ends_at: Date | string | null;
  manual_review: boolean;
  ranking_visible: boolean;
  scoring_rule: ScoringRule;
};

export type SelectedExamProblem = {
  problem_id: number;
  title: string;
  source: string;
  score: number;
  label: string;
  release_after_exam?: boolean;
};

export type ProblemChoice = {
  value: number;
  label: string;
  title: string;
  source: string;
  difficulty?: string;
  tags?: string[];
};

export type ExamBuilderResources = {
  courses: CourseSummary[];
  classes: ClassContext[];
  problems: Problem[];
  preparedProblems: PreparedProblem[];
};

export type CreateExamInput = {
  course_id?: number;
  class_id: number | null;
  title: string;
  description: string;
  starts_at: Date | string | null;
  ends_at: Date | string | null;
  manual_review: boolean;
  ranking_visible: boolean;
  scoring_rule: ScoringRule;
  problems: Array<{
    problem_id: number;
    score: number;
    label: string;
    release_after_exam: boolean;
  }>;
};
