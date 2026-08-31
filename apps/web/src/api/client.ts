import axios from 'axios'

export const apiBase = import.meta.env.VITE_API_BASE || '/api'

export const client = axios.create({
  baseURL: apiBase,
  timeout: 30000,
})

let activeToken = localStorage.getItem('school-oj-token') || ''

export function setActiveToken(token: string) {
  activeToken = token
}

client.interceptors.request.use((config) => {
  const token = activeToken || localStorage.getItem('school-oj-token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export type Role = 'student' | 'teacher' | 'admin'

export interface User {
  id: number
  email: string
  name: string
  role: Role
  student_no?: string
  avatar_url?: string
  can_author?: boolean
  email_verified?: boolean
  account_deleted?: boolean
  created_at?: string
  updated_at?: string
}

export interface Problem {
  id: number
  owner_id?: number
  team_id?: number
  display_code?: string
  title: string
  statement: string
  tags?: Record<string, unknown>
  difficulty?: string
  time_limit_ms: number
  memory_limit_mb: number
  output_limit_kb: number
  manifest?: Record<string, unknown>
  progress_status?: 'unattempted' | 'attempted' | 'accepted'
  points?: number
  points_awarded?: boolean
  deleted_at?: string
}

export type TeamRole = 'owner' | 'admin' | 'member'
export type ScoringRule = 'oi' | 'ioi' | 'acm'

export interface Team {
  id: number
  name: string
  slug: string
  owner_id: number
  owner_name?: string
  visibility: 'private' | 'public'
  join_mode: 'invitation' | 'application' | 'open'
  contest_permission: 'all' | 'admin' | 'owner'
  join_code?: string
  description?: string
  announcement?: string
  icon_url?: string
  member_count?: number
  my_role?: TeamRole
  joined?: boolean
  application_status?: string
  created_at?: string
  updated_at?: string
}

export interface TeamProblemSet {
  id: number
  team_id: number
  title: string
  description?: string
  created_by: number
  problem_count?: number
  can_edit?: boolean
  can_delete?: boolean
  created_at?: string
  updated_at?: string
}

export interface TeamContest {
  id: number
  team_id: number
  title: string
  description?: string
  starts_at?: string
  ends_at?: string
  duration_minutes: number
  scoring_rule?: ScoringRule
  freeze_enabled?: boolean
  freeze_duration_minutes?: number
  gold_award_percent?: number
  silver_award_percent?: number
  bronze_award_percent?: number
  state?: 'draft' | 'published' | 'running' | 'closed'
  created_by: number
  problem_count?: number
  can_edit?: boolean
  can_delete?: boolean
  status?: 'draft' | 'published' | 'running' | 'closed'
  created_at?: string
  updated_at?: string
}

export type ProblemChangeAction = 'create' | 'replace' | 'archive'
export type ProblemChangeStatus = 'pending' | 'processing' | 'completed' | 'rejected' | 'cancelled'

export interface ProblemChangeTicket {
  id: number
  requester_id: number
  requester?: User
  problem_id?: number
  problem?: Problem
  action: ProblemChangeAction
  status: ProblemChangeStatus
  target_scope: 'public' | 'prepared' | 'team_problem_set'
  team_problem_set_id?: number
  description: string
  attachment_name?: string
  resolution_note?: string
  applied_version_id?: number
  processed_by?: number
  processed_at?: string
  created_at: string
  updated_at: string
  impact_summary?: {
    future_exams: number
    pinned_exams: number
    future_contests: number
    pinned_contests: number
    historical_submissions: number
  }
}

export interface ProblemCatalogItem extends Problem {
  pass_rate: number
  accepted_count: number
  evaluated_count: number
}

export interface ProblemCatalogResponse {
  items: ProblemCatalogItem[]
  total: number
  page: number
  page_size: number
  available_tags: string[]
}

export interface PreparedProblem {
  id: number
  problem_id: number
  owner_id: number
  folder?: string
  difficulty?: string
  source?: string
  notes?: string
  archived?: boolean
  published_at?: string
  problem: Problem
  created_at?: string
  updated_at?: string
}

export interface ClassContext {
  id: number
  class_id: number
  class_name: string
  join_code?: string
  archived?: boolean
  course_id: number
  course_code: string
  course_name: string
  term: string
}

export interface Submission {
  id: number
  user_id: number
  problem_id: number
  assignment_id?: number
  exam_id?: number
  team_contest_id?: number
  team_problem_set_id?: number
  language: string
  source_code?: string
  is_public?: boolean
  status: string
  score: number
  manual_score?: number
  manual_graded_by?: number
  manual_graded_at?: string
  time_ms: number
  memory_kb: number
  message: string
  error_point?: string
  user_name?: string
  student_no?: string
  problem_code?: string
  problem_title?: string
  assignment_title?: string
  exam_title?: string
  created_at: string
  updated_at: string
}

export type LatestSubmissionContext =
  | { standalone: true }
  | { assignment_id: number }
  | { exam_id: number }
  | { team_contest_id: number }
  | { problem_set_id: number }

export async function getLatestSubmissions(context: LatestSubmissionContext, problemID?: number) {
  const params: Record<string, string | number | boolean> = { ...context }
  if (problemID) params.problem_id = problemID
  const { data } = await client.get<Submission[]>('/submissions/latest', { params })
  return data || []
}

export interface AuthorApplication {
  id: number
  user_id: number
  motivation: string
  status: 'pending' | 'approved' | 'rejected'
  review_note?: string
  reviewed_by?: number
  reviewed_at?: string
  user?: User
  created_at: string
  updated_at: string
}

export interface ProblemReview {
  id: number
  problem_id: number
  author_id: number
  status: 'pending' | 'approved' | 'rejected' | 'withdrawn'
  review_note?: string
  reviewed_by?: number
  reviewed_at?: string
  submitted_at: string
  test_point_count: number
  problem: Problem
  author: User
  created_at: string
  updated_at: string
}

type SSEListener = (event: MessageEvent<string>) => void

export class AuthenticatedEventSource {
  private readonly controller = new AbortController()
  private readonly listeners = new Map<string, Set<SSEListener>>()

  constructor(path: string) {
    void this.connect(path)
  }

  addEventListener(type: string, listener: SSEListener) {
    const listeners = this.listeners.get(type) || new Set<SSEListener>()
    listeners.add(listener)
    this.listeners.set(type, listeners)
  }

  close() {
    this.controller.abort()
    this.listeners.clear()
  }

  private emit(type: string, data: string) {
    const event = new MessageEvent(type, { data })
    for (const listener of this.listeners.get(type) || []) listener(event)
  }

  private async connect(path: string) {
    const token = activeToken || localStorage.getItem('school-oj-token') || ''
    try {
      const response = await fetch(`${apiBase}${path}`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        signal: this.controller.signal,
      })
      if (!response.ok || !response.body)
        throw new Error(`event stream failed: ${response.status}`)
      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      while (!this.controller.signal.aborted) {
        const { value, done } = await reader.read()
        buffer += decoder.decode(value || new Uint8Array(), { stream: !done })
        buffer = buffer.replace(/\r\n/g, '\n')
        let boundary = buffer.indexOf('\n\n')
        while (boundary >= 0) {
          const frame = buffer.slice(0, boundary)
          buffer = buffer.slice(boundary + 2)
          this.consumeFrame(frame)
          boundary = buffer.indexOf('\n\n')
        }
        if (done) break
      }
    } catch (error: any) {
      if (!this.controller.signal.aborted) {
        this.emit(
          'error',
          JSON.stringify({ error: error?.message || 'event stream failed' }),
        )
      }
    }
  }

  private consumeFrame(frame: string) {
    let type = 'message'
    const data: string[] = []
    for (const line of frame.split('\n')) {
      if (line.startsWith('event:')) type = line.slice(6).trim()
      if (line.startsWith('data:')) data.push(line.slice(5).trimStart())
    }
    if (data.length) this.emit(type, data.join('\n'))
  }
}

export function openEventStream(path: string) {
  return new AuthenticatedEventSource(path)
}
