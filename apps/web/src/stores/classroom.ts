import { defineStore } from 'pinia'
import { client, type ClassContext } from '../api/client'

export const useClassroomStore = defineStore('classroom', {
  state: () => ({
    classes: [] as ClassContext[],
    activeClassId: Number(localStorage.getItem('school-oj-active-class') || 0),
    loading: false
  }),
  getters: {
    activeClass: (state) => state.classes.find((item) => item.class_id === state.activeClassId) || null
  },
  actions: {
    async load() {
      this.loading = true
      try {
        const { data } = await client.get('/me/classes')
        this.classes = data
        if (!this.classes.some((item) => item.class_id === this.activeClassId)) {
          this.activeClassId = this.classes[0]?.class_id || 0
        }
        this.persist()
      } finally {
        this.loading = false
      }
    },
    setActive(classID: number) {
      this.activeClassId = classID
      this.persist()
    },
    clear() {
      this.classes = []
      this.activeClassId = 0
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
