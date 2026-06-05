import { defineStore } from 'pinia'
import { client } from '../api/client'

const storageKey = 'school-oj-active-exam'

type ActiveExam = {
  id: number
  title?: string
  starts_at?: string
  ends_at?: string
}

function activeExamMessage(title?: string) {
  return title ? `你有正在进行的考试「${title}」，请及时返回考试界面。` : '你有正在进行的考试，请及时返回考试界面。'
}

export const useExamLockStore = defineStore('examLock', {
  state: () => ({
    locked: false,
    examId: undefined as number | undefined,
    title: '',
    message: '你有正在进行的考试，请及时返回考试界面。',
    synced: false
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
          this.title = value.title || ''
          this.message = value.message || activeExamMessage(this.title)
        }
      } catch {
        localStorage.removeItem(storageKey)
      }
    },
    lock(examId?: number, message?: string, title?: string) {
      this.locked = true
      if (examId) this.examId = examId
      if (title !== undefined) this.title = title
      this.message = message || activeExamMessage(this.title)
      localStorage.setItem(storageKey, JSON.stringify({ examId: this.examId, title: this.title, message: this.message }))
    },
    unlock() {
      this.locked = false
      this.examId = undefined
      this.title = ''
      this.message = activeExamMessage()
      localStorage.removeItem(storageKey)
    },
    async syncActiveExam() {
      try {
        const { data } = await client.get('/me/active-exam')
        const exam = data?.exam as ActiveExam | undefined
        if (data?.active && exam?.id) {
          this.lock(exam.id, activeExamMessage(exam.title), exam.title || '')
        } else {
          this.unlock()
        }
      } finally {
        this.synced = true
      }
      return { active: this.locked, examId: this.examId, title: this.title }
    }
  }
})
