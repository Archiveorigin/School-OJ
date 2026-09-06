import { onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { createExamActivityQueue, ExamFocusState, isDevtoolsShortcut, type ExamActivityQueue } from './activity'

export function useExamEnvironment(active: Ref<boolean>, userID: Ref<number>, examID: Ref<number>) {
  const warning = ref(false)
  let queue: ExamActivityQueue | undefined
  let timer: number | undefined
  let lastShortcut = 0
  let state = new ExamFocusState(event => { queue?.push(event); if (event === 'window_leave') warning.value = true; void queue?.flush(document.hidden) })
  const blur = () => state.update(!document.hidden, false)
  const focus = () => state.update(!document.hidden, true)
  const visibility = () => state.update(!document.hidden, document.hasFocus())
  const online = () => void queue?.flush()
  const pagehide = () => { blur(); void queue?.flush(true) }
  const keydown = (event: KeyboardEvent) => {
    if (!isDevtoolsShortcut(event)) return
    event.preventDefault(); event.stopImmediatePropagation(); warning.value = true
    if (Date.now() - lastShortcut > 2000) { queue?.push('devtools_shortcut'); void queue?.flush(); lastShortcut = Date.now() }
  }
  function stop() {
    window.removeEventListener('blur', blur); window.removeEventListener('focus', focus)
    window.removeEventListener('online', online); window.removeEventListener('pagehide', pagehide)
    document.removeEventListener('visibilitychange', visibility); window.removeEventListener('keydown', keydown, true)
    if (timer) window.clearInterval(timer)
    void queue?.flush(true); queue = undefined; warning.value = false
  }
  watch([active, userID, examID], ([enabled, uid, id]) => {
    stop()
    if (!enabled || !uid || !id) return
    queue = createExamActivityQueue(uid, id)
    state = new ExamFocusState(event => { queue?.push(event); if (event === 'window_leave') warning.value = true; void queue?.flush(document.hidden) })
    window.addEventListener('blur', blur); window.addEventListener('focus', focus)
    window.addEventListener('online', online); window.addEventListener('pagehide', pagehide)
    document.addEventListener('visibilitychange', visibility); window.addEventListener('keydown', keydown, true)
    timer = window.setInterval(online, 15000)
    void queue.flush()
  }, { immediate: true })
  onBeforeUnmount(stop)
  return { warning, flush: () => queue?.flush() }
}
