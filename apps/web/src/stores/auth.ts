import { defineStore } from 'pinia'
import { client, type User } from '../api/client'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('school-oj-token') || '',
    user: JSON.parse(localStorage.getItem('school-oj-user') || 'null') as User | null
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
      localStorage.setItem('school-oj-token', data.token)
      localStorage.setItem('school-oj-user', JSON.stringify(data.user))
    },
    logout() {
      this.token = ''
      this.user = null
      localStorage.removeItem('school-oj-token')
      localStorage.removeItem('school-oj-user')
    }
  }
})
