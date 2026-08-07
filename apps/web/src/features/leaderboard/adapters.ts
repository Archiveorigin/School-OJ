import type {
  LeaderboardData,
  LeaderboardProblem,
  LeaderboardResult,
  LeaderboardRow,
  LeaderboardStatus
} from './types'

const problemColors = [
  '#ff46a0', '#ff7a45', '#f4c430', '#46b96b', '#23b7c8', '#3b82f6', '#6f76d9',
  '#9b5de5', '#ef476f', '#06d6a0', '#118ab2', '#8b5cf6', '#ec4899'
]

type UnknownRecord = Record<string, any>

function safeNumber(value: unknown, fallback = 0) {
  const number = Number(value)
  return Number.isFinite(number) ? number : fallback
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

function problemList(items: UnknownRecord[] = []): LeaderboardProblem[] {
  return items.map((problem, index) => ({
    id: problem.problem_id ?? problem.id ?? index,
    label: String(problem.label || problem.display_code || String.fromCharCode(65 + index)),
    title: problem.title ? String(problem.title) : undefined,
    color: problemColors[index % problemColors.length]
  }))
}

function teamStatus(cell: UnknownRecord | undefined): LeaderboardStatus {
  if (!cell || !safeNumber(cell.attempts)) return 'none'
  if (cell.status === 'accepted') return 'accepted'
  if (cell.status === 'queued' || cell.status === 'running') return 'pending'
  return 'wrong'
}

function examStatus(cell: UnknownRecord | undefined): LeaderboardStatus {
  if (!cell || (!cell.status && !cell.submitted_at)) return 'none'
  if (cell.pending || !cell.score_ready) return 'pending'
  if (safeNumber(cell.max_score) > 0 && safeNumber(cell.best_score) >= safeNumber(cell.max_score)) return 'accepted'
  return 'wrong'
}

export function adaptTeamContestRanking(source: UnknownRecord | null | undefined, fallback: UnknownRecord = {}): LeaderboardData {
  const contest = source?.contest || fallback.contest || {}
  const scoringRule = String(source?.scoring_rule || contest.scoring_rule || fallback.scoring_rule || 'penalty')
  const problems = problemList(source?.problems || [])
  const durationSeconds = Math.max(0, Math.round(safeNumber(contest.duration_minutes, 0) * 60))
  const rows: LeaderboardRow[] = (source?.rows || []).map((row: UnknownRecord, index: number) => {
    const rawResults = new Map<string, UnknownRecord>((row.problems || []).map((cell: UnknownRecord) => [String(cell.problem_id), cell]))
    const results: Record<string, LeaderboardResult> = {}
    for (const problem of problems) {
      const cell = rawResults.get(String(problem.id))
      const attempts = safeNumber(cell?.attempts)
      const elapsedMinutes = safeNumber(cell?.elapsed_minutes)
      results[String(problem.id)] = {
        status: teamStatus(cell),
        attempts,
        timeSeconds: attempts ? elapsedMinutes * 60 : undefined,
        firstBlood: Boolean(cell?.fastest),
        primary: attempts ? String(attempts) : undefined,
        secondary: attempts ? `${elapsedMinutes}'` : undefined
      }
    }
    const metric = scoringRule === 'score' ? safeNumber(row.total_score) : safeNumber(row.penalty_minutes)
    return {
      id: row.user_id ?? index,
      rank: index + 1,
      name: String(row.name || `参赛者 ${index + 1}`),
      organization: String(fallback.team?.name || fallback.team_name || '团队成员'),
      meta: `提交 ${safeNumber(row.submission_count)} 次`,
      solved: safeNumber(row.solved),
      metric,
      metricDisplay: scoringRule === 'score' ? String(metric) : `${metric}'`,
      submissions: safeNumber(row.submission_count),
      results
    }
  })
  return {
    title: String(contest.title || fallback.title || '团队比赛实时榜单'),
    subtitle: scoringRule === 'score' ? '通过数优先 · 总分排名' : '通过数优先 · 罚时排名',
    durationSeconds,
    currentTimeSeconds: currentTimeFromRange(contest.starts_at, durationSeconds),
    identityLabel: '参赛者 / 团队',
    solvedLabel: '通过',
    metricLabel: scoringRule === 'score' ? '总分' : '罚时',
    metricDirection: scoringRule === 'score' ? 'descending' : 'ascending',
    problems,
    rows
  }
}

export function adaptExamRanking(source: UnknownRecord | null | undefined): LeaderboardData {
  const exam = source?.exam || {}
  const problems = problemList(source?.problems || [])
  const durationSeconds = durationFromRange(exam.starts_at, exam.ends_at)
  const rows: LeaderboardRow[] = (source?.rows || []).map((row: UnknownRecord, index: number) => {
    const rawResults = new Map<string, UnknownRecord>((row.problems || []).map((cell: UnknownRecord) => [String(cell.problem_id), cell]))
    const results: Record<string, LeaderboardResult> = {}
    for (const problem of problems) {
      const cell = rawResults.get(String(problem.id))
      const status = examStatus(cell)
      const bestScore = safeNumber(cell?.best_score)
      const maxScore = safeNumber(cell?.max_score)
      results[String(problem.id)] = {
        status,
        attempts: status === 'none' ? 0 : 1,
        timeSeconds: elapsedFromStart(exam.starts_at, cell?.submitted_at),
        primary: status === 'none' ? undefined : cell?.score_ready ? `${bestScore}/${maxScore}` : '…',
        secondary: status === 'pending' ? '待评分' : status === 'accepted' ? '满分' : status === 'wrong' ? '未满分' : undefined
      }
    }
    const totalScore = safeNumber(row.total_score)
    const maxScore = safeNumber(row.max_score, safeNumber(source?.stats?.max_score))
    return {
      id: row.user_id ?? index,
      rank: safeNumber(row.rank, index + 1),
      name: String(row.name || `学生 ${index + 1}`),
      organization: row.student_no ? `学号 ${row.student_no}` : String(exam.class_name || '课程学生'),
      meta: `提交 ${safeNumber(row.submission_count)} 次${safeNumber(row.pending_count) ? ` · ${safeNumber(row.pending_count)} 项待评分` : ''}`,
      solved: safeNumber(row.solved),
      metric: totalScore,
      metricDisplay: `${totalScore}/${maxScore}`,
      submissions: safeNumber(row.submission_count),
      results
    }
  })
  return {
    title: String(exam.title || '考试实时榜单'),
    subtitle: [exam.course_name, exam.class_name].filter(Boolean).join(' · ') || '课程考试',
    durationSeconds,
    currentTimeSeconds: currentTimeFromRange(exam.starts_at, durationSeconds, source?.now),
    identityLabel: '学生 / 班级',
    solvedLabel: '满分',
    metricLabel: '总分',
    metricDirection: 'descending',
    problems,
    rows
  }
}
