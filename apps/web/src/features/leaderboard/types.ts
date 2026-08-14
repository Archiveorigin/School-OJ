export type LeaderboardStatus = 'accepted' | 'wrong' | 'pending' | 'frozen' | 'none'
export type LeaderboardMetricDirection = 'ascending' | 'descending'
export type LeaderboardScoringRule = 'penalty' | 'score'
export type LeaderboardAwardTier = 'gold' | 'silver' | 'bronze' | 'none'

export interface LeaderboardAwardPercents {
  gold: number
  silver: number
  bronze: number
}

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
  solvedLabel: string
  metricLabel: string
  metricDirection: LeaderboardMetricDirection
  participantCount?: number
  awardPercents?: LeaderboardAwardPercents
  problems: LeaderboardProblem[]
  rows: LeaderboardRow[]
}
