import { defineStore } from 'pinia'
import { client, type User } from '../api/client'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('school-oj-token') || '',
    user: JSON.parse(localStorage.getItem('school-oj-user') || 'null') as User | null,
    hydrated: false,
    hydratePromise: null as Promise<void> | null
  }),
  getters: {
    isAuthed: (state) => Boolean(state.token),
    role: (state) => state.user?.role
  },
  actions: {
    async login(email: string, password: string) {
      const { data } = await client.post('/auth/login', { email, password })
      this.token = data.token
      this.user = data.user
      this.hydrated = true
      localStorage.setItem('school-oj-token', data.token)
      localStorage.setItem('school-oj-user', JSON.stringify(data.user))
    },
    async hydrate() {
      if (this.hydratePromise) return this.hydratePromise
      if (!this.token) {
        this.user = null
        this.hydrated = true
        localStorage.removeItem('school-oj-user')
        return
      }
      this.hydratePromise = (async () => {
        try {
          const { data } = await client.get('/me')
          this.user = data
          localStorage.setItem('school-oj-user', JSON.stringify(data))
        } catch {
          this.logout()
        } finally {
          this.hydrated = true
          this.hydratePromise = null
        }
      })()
      return this.hydratePromise
    },
    logout() {
      this.token = ''
      this.user = null
      this.hydrated = true
      this.hydratePromise = null
      localStorage.removeItem('school-oj-token')
      localStorage.removeItem('school-oj-user')
    }
  }
})
