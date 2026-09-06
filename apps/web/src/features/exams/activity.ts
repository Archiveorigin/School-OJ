import { apiBase, generatedApi, apiOperations, type components } from '../../api/client'
import { getActiveToken } from '../../api/http'

export type ExamClientEvent = 'window_leave' | 'window_return' | 'devtools_shortcut'
export type ExamActivityEvent = components['schemas']['ExamActivityEvent']
const prefix = 'school-oj-exam-activity:'
const limit = 200

export function activityID() {
  const bytes = crypto.getRandomValues(new Uint8Array(16))
  return Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('')
}

export class ExamFocusState {
  away = false
  constructor(private emit: (event: ExamClientEvent) => void) {}
  update(visible: boolean, focused: boolean) {
    const away = !visible || !focused
    if (away === this.away) return
    this.away = away
    this.emit(away ? 'window_leave' : 'window_return')
  }
}

export function isDevtoolsShortcut(event: Pick<KeyboardEvent, 'key' | 'ctrlKey' | 'metaKey' | 'shiftKey' | 'altKey'>) {
  const key = event.key.toLowerCase()
  return key === 'f12' || (event.ctrlKey && event.shiftKey && ['i', 'j', 'c'].includes(key)) || (event.metaKey && event.altKey && ['i', 'j', 'c'].includes(key))
}

export class ExamActivityQueue {
  private events: ExamActivityEvent[] = []
  private flushing = false
  constructor(readonly examID: number, readonly storageKey: string, private storage: Storage = localStorage) {
    try { this.events = JSON.parse(storage.getItem(storageKey) || '[]').filter((item: ExamActivityEvent) => item.id && item.client_at && ['window_leave', 'window_return', 'devtools_shortcut'].includes(item.event)).slice(-limit) }
    catch { this.events = [] }
  }
  push(event: ExamClientEvent) {
    this.events.push({ id: activityID(), event, client_at: new Date().toISOString() })
    this.events = this.events.slice(-limit)
    this.persist()
  }
  private persist() { try { if (this.events.length) this.storage.setItem(this.storageKey, JSON.stringify(this.events)); else this.storage.removeItem(this.storageKey) } catch { /* In-memory retry remains available if browser storage is full. */ } }
  async flush(keepalive = false) {
    if (this.flushing || !this.events.length || !getActiveToken()) return
    this.flushing = true
    const batch = this.events.slice(0, 30)
    try {
      if (keepalive) {
        const endpoint = apiOperations.recordExamActivity.path.replace('{id}', String(this.examID))
        const response = await fetch(`${apiBase}${endpoint}`, { method: 'POST', headers: { Authorization: `Bearer ${getActiveToken()}`, 'Content-Type': 'application/json' }, body: JSON.stringify({ events: batch }), keepalive: true })
        if (!response.ok) return
      } else await generatedApi.recordExamActivity({ path: { id: this.examID }, body: { events: batch } })
      const ids = new Set(batch.map(event => event.id))
      this.events = this.events.filter(event => !ids.has(event.id))
      this.persist()
    } catch { /* Retain the bounded queue for reconnect/reload. */ }
    finally { this.flushing = false }
  }
}

export function createExamActivityQueue(userID: number, examID: number) {
  let tab = sessionStorage.getItem('school-oj-activity-tab')
  if (!tab) { tab = activityID(); sessionStorage.setItem('school-oj-activity-tab', tab) }
  return new ExamActivityQueue(examID, `${prefix}${userID}:${examID}:${tab}`)
}

export async function flushSavedExamActivity(userID: number) {
  for (const key of Object.keys(localStorage)) {
    if (!key.startsWith(`${prefix}${userID}:`)) continue
    const examID = Number(key.split(':')[2])
    if (examID) await new ExamActivityQueue(examID, key).flush()
  }
}
