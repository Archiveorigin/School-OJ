export type LeaderboardStatus = 'accepted' | 'wrong' | 'pending' | 'frozen' | 'none'
export type LeaderboardMetricDirection = 'ascending' | 'descending'

export interface LeaderboardProblem {
  id: number | string
  label: string
  title?: string
  color: string
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
  organization?: string
  meta?: string
  solved: number
  metric: number
  metricDisplay?: string
  submissions: number
  results: Record<string, LeaderboardResult>
}

export interface LeaderboardData {
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
