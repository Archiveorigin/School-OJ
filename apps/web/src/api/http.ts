import axios from 'axios'

export const apiBase = import.meta.env.VITE_API_BASE || '/api/v1'

export const http = axios.create({
  baseURL: apiBase,
  timeout: 30000,
})

let activeToken = localStorage.getItem('school-oj-token') || ''

export function setActiveToken(token: string) {
  activeToken = token
}

export function getActiveToken() {
  return activeToken || localStorage.getItem('school-oj-token') || ''
}

http.interceptors.request.use((config) => {
  const token = getActiveToken()
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})
