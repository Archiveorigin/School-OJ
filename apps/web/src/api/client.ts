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
}

export interface Problem {
  id: number
  slug: string
  title: string
  statement: string
  time_limit_ms: number
  memory_limit_mb: number
  output_limit_kb: number
}

export interface Submission {
  id: number
  user_id: number
  problem_id: number
  language: string
  status: string
  score: number
  time_ms: number
  memory_kb: number
  message: string
  created_at: string
  updated_at: string
}

export function sseUrl(path: string) {
  const token = localStorage.getItem('school-oj-token')
  const sep = path.includes('?') ? '&' : '?'
  return `${apiBase}${path}${sep}token=${encodeURIComponent(token || '')}`
}
