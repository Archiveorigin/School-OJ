import axios from 'axios'

export const apiBase = import.meta.env.VITE_API_BASE || '/api'

export const client = axios.create({
  baseURL: apiBase,
  timeout: 30000
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
  email_verified?: boolean
  account_deleted?: boolean
  created_at?: string
  updated_at?: string
}

export interface Problem {
  id: number
  owner_id?: number
  display_code?: string
  slug: string
  title: string
  statement: string
  tags?: Record<string, unknown>
  time_limit_ms: number
  memory_limit_mb: number
  output_limit_kb: number
  manifest?: Record<string, unknown>
  progress_status?: 'unattempted' | 'attempted' | 'accepted'
  points?: number
  points_awarded?: boolean
  deleted_at?: string
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
  language: string
  source_code?: string
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

export function sseUrl(path: string) {
  const token = localStorage.getItem('school-oj-token')
  const sep = path.includes('?') ? '&' : '?'
  return `${apiBase}${path}${sep}token=${encodeURIComponent(token || '')}`
}

export function problemAssetUrl(problemID: number, path: string) {
  const token = localStorage.getItem('school-oj-token')
  const clean = path.replace(/^\/+/, '')
  const encoded = clean
    .split('/')
    .map((part) => encodeURIComponent(part))
    .join('/')
  return `${apiBase}/problems/${problemID}/assets/${encoded}?token=${encodeURIComponent(token || '')}`
}
