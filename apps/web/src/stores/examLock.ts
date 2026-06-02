import { defineStore } from 'pinia'

export const useExamLockStore = defineStore('examLock', {
  state: () => ({
    locked: false,
    message: '该考试要求提交所有题目后退出'
  }),
  actions: {
    lock(message?: string) {
      this.locked = true
      if (message) this.message = message
    },
    unlock() {
      this.locked = false
    }
  }
})
