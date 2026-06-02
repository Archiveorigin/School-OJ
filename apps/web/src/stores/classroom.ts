import { defineStore } from 'pinia'
import { client, type ClassContext } from '../api/client'

export const useClassroomStore = defineStore('classroom', {
  state: () => ({
    classes: [] as ClassContext[],
    activeClassId: Number(localStorage.getItem('school-oj-active-class') || 0),
    loading: false,
    loaded: false,
    loadPromise: null as Promise<void> | null
  }),
  getters: {
    activeClass: (state) => state.classes.find((item) => item.class_id === state.activeClassId) || null
  },
  actions: {
    async load(options: { force?: boolean } = {}) {
      if (this.loadPromise && !options.force) return this.loadPromise
      this.loading = true
      this.loadPromise = (async () => {
        try {
          const { data } = await client.get('/me/classes')
          this.classes = Array.isArray(data) ? data : []
          if (!this.classes.some((item) => item.class_id === this.activeClassId)) {
            this.activeClassId = this.classes[0]?.class_id || 0
          }
          this.loaded = true
          this.persist()
        } finally {
          this.loading = false
          this.loadPromise = null
        }
      })()
      return this.loadPromise
    },
    setActive(classID: number) {
      this.activeClassId = classID
      this.persist()
    },
    clear() {
      this.classes = []
      this.activeClassId = 0
      this.loading = false
      this.loaded = false
      this.loadPromise = null
      localStorage.removeItem('school-oj-active-class')
    },
    persist() {
      if (this.activeClassId) {
        localStorage.setItem('school-oj-active-class', String(this.activeClassId))
      } else {
        localStorage.removeItem('school-oj-active-class')
      }
    }
  }
})
