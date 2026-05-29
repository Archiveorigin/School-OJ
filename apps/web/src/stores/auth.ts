import { defineStore } from 'pinia'
import { client, setActiveToken, type User } from '../api/client'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('school-oj-token') || '',
    user: JSON.parse(localStorage.getItem('school-oj-user') || 'null') as User | null,
    theme: localStorage.getItem('huanghai-theme') || 'light',
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
      setActiveToken(data.token)
      localStorage.setItem('school-oj-token', data.token)
      localStorage.setItem('school-oj-user', JSON.stringify(data.user))
    },
    setSession(token: string, user: User) {
      this.token = token
      this.user = user
      this.hydrated = true
      setActiveToken(token)
      localStorage.setItem('school-oj-token', token)
      localStorage.setItem('school-oj-user', JSON.stringify(user))
    },
    updateUser(user: User) {
      this.user = user
      localStorage.setItem('school-oj-user', JSON.stringify(user))
    },
    async hydrate() {
      if (this.hydratePromise) return this.hydratePromise
      if (!this.token) {
        this.user = null
        this.hydrated = true
        setActiveToken('')
        localStorage.removeItem('school-oj-user')
        return
      }
      setActiveToken(this.token)
      this.hydratePromise = (async () => {
        try {
          const { data } = await client.get('/me')
          this.user = data
          setActiveToken(this.token)
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
      setActiveToken('')
      localStorage.removeItem('school-oj-token')
      localStorage.removeItem('school-oj-user')
    },
    applyTheme() {
      document.documentElement.classList.toggle('dark', this.theme === 'dark')
      localStorage.setItem('huanghai-theme', this.theme)
    },
    toggleTheme() {
      this.theme = this.theme === 'dark' ? 'light' : 'dark'
      this.applyTheme()
    }
  }
})
