export type LeaderboardStatus = 'accepted' | 'wrong' | 'pending' | 'frozen' | 'none'
export type LeaderboardMetricDirection = 'ascending' | 'descending'
export type LeaderboardScoringRule = 'penalty' | 'score'

export interface LeaderboardProblem {
  id: number | string
  label: string
  title?: string
  color: string
  maxScore: number
}

export interface LeaderboardResult {
  status: LeaderboardStatus
  attempts: number
  timeSeconds?: number
  firstBlood?: boolean
  primary?: string
  secondary?: string
}

export interface LeaderboardRow {
  id: number | string
  rank: number
  name: string
  studentNo: string
  meta?: string
  solved: number
  metric: number
  metricDisplay?: string
  maxScore?: number
  submissions: number
  results: Record<string, LeaderboardResult>
}

export interface LeaderboardData {
  scoringRule: LeaderboardScoringRule
  title: string
  subtitle?: string
  durationSeconds: number
  currentTimeSeconds: number
  identityLabel: string
  solvedLabel: string
  metricLabel: string
  metricDirection: LeaderboardMetricDirection
  problems: LeaderboardProblem[]
  rows: LeaderboardRow[]
}
