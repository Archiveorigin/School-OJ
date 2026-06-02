import { defineStore } from 'pinia'

const storageKey = 'school-oj-locked-exam'

export const useExamLockStore = defineStore('examLock', {
  state: () => ({
    locked: false,
    examId: undefined as number | undefined,
    message: '该考试要求提交所有题目后退出'
  }),
  actions: {
    hydrate() {
      const raw = localStorage.getItem(storageKey)
      if (!raw) return
      try {
        const value = JSON.parse(raw)
        if (value?.examId) {
          this.locked = true
          this.examId = Number(value.examId)
          this.message = value.message || this.message
        }
      } catch {
        localStorage.removeItem(storageKey)
      }
    },
    lock(examId?: number, message?: string) {
      this.locked = true
      if (examId) this.examId = examId
      if (message) this.message = message
      localStorage.setItem(storageKey, JSON.stringify({ examId: this.examId, message: this.message }))
    },
    unlock() {
      this.locked = false
      this.examId = undefined
      localStorage.removeItem(storageKey)
    }
  }
})
