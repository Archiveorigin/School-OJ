import type {
  LeaderboardData,
  LeaderboardProblem,
  LeaderboardResult,
  LeaderboardRow,
  LeaderboardScoringRule,
  LeaderboardStatus
} from './types'
import { normalizeAwardPercents } from './awards'

const problemColors = [
  '#ff46a0', '#ff7a45', '#f4c430', '#46b96b', '#23b7c8', '#3b82f6', '#6f76d9',
  '#9b5de5', '#ef476f', '#06d6a0', '#118ab2', '#8b5cf6', '#ec4899'
]

type UnknownRecord = Record<string, any>

function safeNumber(value: unknown, fallback = 0) {
  const number = Number(value)
  return Number.isFinite(number) ? number : fallback
}

function scoringRule(value: unknown): LeaderboardScoringRule {
  return value === 'score' ? 'score' : 'penalty'
}

function parseTime(value: unknown) {
  if (!value) return undefined
  const milliseconds = new Date(String(value)).getTime()
  return Number.isFinite(milliseconds) ? milliseconds : undefined
}

function durationFromRange(startsAt: unknown, endsAt: unknown, fallbackSeconds = 0) {
  const start = parseTime(startsAt)
  const end = parseTime(endsAt)
  if (start !== undefined && end !== undefined && end >= start) return Math.round((end - start) / 1000)
  return Math.max(0, Math.round(fallbackSeconds))
}

function currentTimeFromRange(startsAt: unknown, durationSeconds: number, nowValue?: unknown) {
  const start = parseTime(startsAt)
  if (start === undefined) return durationSeconds
  const now = parseTime(nowValue) ?? Date.now()
  return Math.min(durationSeconds, Math.max(0, Math.round((now - start) / 1000)))
}

function elapsedFromStart(startsAt: unknown, submittedAt: unknown) {
  const start = parseTime(startsAt)
  const submitted = parseTime(submittedAt)
  if (start === undefined || submitted === undefined) return undefined
  return Math.max(0, Math.round((submitted - start) / 1000))
}

function problemList(items: UnknownRecord[] = [], defaultMaxScore = 100): LeaderboardProblem[] {
  return items.map((problem, index) => ({
    id: problem.problem_id ?? problem.id ?? index,
    label: String(problem.label || problem.display_code || String.fromCharCode(65 + index)),
    title: problem.title ? String(problem.title) : undefined,
    color: problemColors[index % problemColors.length],
    maxScore: Math.max(0, safeNumber(problem.score, defaultMaxScore))
  }))
}

function penaltyStatus(cell: UnknownRecord | undefined): LeaderboardStatus {
  if (!cell || !safeNumber(cell.attempts)) return 'none'
  if (cell.solved_at || cell.status === 'accepted') return 'accepted'
  if (cell.pending || cell.status === 'queued' || cell.status === 'running') return 'pending'
  return 'wrong'
}

function scoreStatus(cell: UnknownRecord | undefined, maxScore: number): LeaderboardStatus {
  if (!cell || (!cell.status && !cell.submitted_at && !safeNumber(cell.attempts))) return 'none'
  if (maxScore > 0 && safeNumber(cell.best_score) >= maxScore) return 'accepted'
  const awaitingManualScore = cell.pending === true || cell.score_ready === false
  if (awaitingManualScore || cell.status === 'queued' || cell.status === 'running') return 'pending'
  return 'wrong'
}

export function adaptTeamContestRanking(source: UnknownRecord | null | undefined, fallback: UnknownRecord = {}): LeaderboardData {
  const contest = source?.contest || fallback.contest || {}
  const rule = scoringRule(source?.scoring_rule ?? contest.scoring_rule ?? fallback.scoring_rule)
  const problems = problemList(source?.problems || [], 100)
  const durationSeconds = Math.max(0, Math.round(safeNumber(contest.duration_minutes, 0) * 60))
  const rows: LeaderboardRow[] = (source?.rows || []).map((row: UnknownRecord, index: number) => {
    const rawResults = new Map<string, UnknownRecord>((row.problems || []).map((cell: UnknownRecord) => [String(cell.problem_id), cell]))
    const results: Record<string, LeaderboardResult> = {}
    for (const problem of problems) {
      const cell = rawResults.get(String(problem.id))
      const attempts = safeNumber(cell?.attempts)
      const elapsedMinutes = safeNumber(cell?.elapsed_minutes)
      if (rule === 'score') {
        const bestScore = safeNumber(cell?.best_score)
        const status = scoreStatus(cell, problem.maxScore)
        results[String(problem.id)] = {
          status,
          attempts,
          primary: status === 'none' ? undefined : `${bestScore}/${problem.maxScore}`,
          secondary: status === 'pending' ? '待评分' : status === 'accepted' ? '满分' : status === 'wrong' ? '未满分' : undefined
        }
      } else {
        results[String(problem.id)] = {
          status: penaltyStatus(cell),
          attempts,
          timeSeconds: attempts ? elapsedMinutes * 60 : undefined,
          firstBlood: Boolean(cell?.fastest),
          primary: attempts ? String(attempts) : undefined,
          secondary: attempts ? `${elapsedMinutes}'` : undefined
        }
      }
    }
    const metric = rule === 'score' ? safeNumber(row.total_score) : safeNumber(row.penalty_minutes)
    const maxScore = rule === 'score'
      ? safeNumber(row.max_score, problems.reduce((sum, problem) => sum + problem.maxScore, 0))
      : undefined
    return {
      id: row.user_id ?? index,
      rank: safeNumber(row.rank, index + 1),
      name: String(row.name || `参赛者 ${index + 1}`),
      solved: safeNumber(row.solved),
      metric,
      metricDisplay: rule === 'score' ? `${metric}/${maxScore}` : String(metric),
      maxScore,
      submissions: safeNumber(row.submission_count),
      results
    }
  })
  const awardSource = source?.award_percentages || source?.awards || contest
  const awardPercents = normalizeAwardPercents({
    gold: awardSource?.gold_award_percent,
    silver: awardSource?.silver_award_percent,
    bronze: awardSource?.bronze_award_percent
  })
  return {
    scoringRule: rule,
    title: String(contest.title || fallback.title || '团队比赛实时榜单'),
    subtitle: rule === 'score' ? '总分优先 · 满分题数次序' : '题数优先 · 罚时排名',
    durationSeconds,
    currentTimeSeconds: currentTimeFromRange(contest.starts_at, durationSeconds),
    solvedLabel: rule === 'score' ? '满分' : '题数',
    metricLabel: rule === 'score' ? '总分' : '罚时',
    metricDirection: rule === 'score' ? 'descending' : 'ascending',
    participantCount: Math.max(rows.length, safeNumber(source?.participant_count ?? source?.total_participants, rows.length)),
    awardPercents,
    problems,
    rows
  }
}

export function adaptExamRanking(source: UnknownRecord | null | undefined): LeaderboardData {
  const exam = source?.exam || {}
  const rule = scoringRule(source?.scoring_rule ?? exam.scoring_rule)
  const problems = problemList(source?.problems || [], 100)
  const durationSeconds = durationFromRange(exam.starts_at, exam.ends_at)
  const rows: LeaderboardRow[] = (source?.rows || []).map((row: UnknownRecord, index: number) => {
    const rawResults = new Map<string, UnknownRecord>((row.problems || []).map((cell: UnknownRecord) => [String(cell.problem_id), cell]))
    const results: Record<string, LeaderboardResult> = {}
    for (const problem of problems) {
      const cell = rawResults.get(String(problem.id))
      const attempts = safeNumber(cell?.attempts, cell?.status || cell?.submitted_at ? 1 : 0)
      if (rule === 'score') {
        const maxScore = safeNumber(cell?.max_score, problem.maxScore)
        const bestScore = safeNumber(cell?.best_score)
        const status = scoreStatus(cell, maxScore)
        results[String(problem.id)] = {
          status,
          attempts,
          timeSeconds: elapsedFromStart(exam.starts_at, cell?.submitted_at),
          primary: status === 'none' ? undefined : `${bestScore}/${maxScore}`,
          secondary: status === 'pending' ? '待评分' : status === 'accepted' ? '满分' : status === 'wrong' ? '未满分' : undefined
        }
      } else {
        const elapsedMinutes = safeNumber(cell?.elapsed_minutes)
        results[String(problem.id)] = {
          status: penaltyStatus(cell),
          attempts,
          timeSeconds: attempts ? elapsedMinutes * 60 : undefined,
          firstBlood: Boolean(cell?.fastest),
          primary: attempts ? String(attempts) : undefined,
          secondary: attempts ? `${elapsedMinutes}'` : undefined
        }
      }
    }
    const totalScore = safeNumber(row.total_score)
    const maxScore = safeNumber(row.max_score, safeNumber(source?.stats?.max_score))
    const metric = rule === 'score' ? totalScore : safeNumber(row.penalty_minutes)
    return {
      id: row.user_id ?? index,
      rank: safeNumber(row.rank, index + 1),
      name: String(row.name || `学生 ${index + 1}`),
      solved: safeNumber(row.solved),
      metric,
      metricDisplay: rule === 'score' ? `${totalScore}/${maxScore}` : String(metric),
      maxScore: rule === 'score' ? maxScore : undefined,
      submissions: safeNumber(row.submission_count),
      results
    }
  })
  return {
    scoringRule: rule,
    title: String(exam.title || '考试实时榜单'),
    subtitle: [exam.course_name, exam.class_name].filter(Boolean).join(' · ') || '课程考试',
    durationSeconds,
    currentTimeSeconds: currentTimeFromRange(exam.starts_at, durationSeconds, source?.now),
    solvedLabel: rule === 'score' ? '满分' : '题数',
    metricLabel: rule === 'score' ? '总分' : '罚时',
    metricDirection: rule === 'score' ? 'descending' : 'ascending',
    participantCount: Math.max(rows.length, safeNumber(source?.participant_count ?? source?.total_participants, rows.length)),
    awardPercents: normalizeAwardPercents(),
    problems,
    rows
  }
}
