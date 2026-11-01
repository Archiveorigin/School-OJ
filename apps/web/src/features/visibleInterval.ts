import { onBeforeUnmount, onMounted, watch } from 'vue'

/**
 * Repeats a task while the page is visible. A hidden tab issues no requests; when it
 * becomes visible again an overdue task runs at once, so the view is never stale.
 * Runs never overlap: the next one is scheduled only after the previous one settles.
 * Becoming enabled does not run the task, because the caller just loaded its view.
 */
export function useVisibleInterval(task: () => unknown, intervalMs: number, enabled: () => boolean = () => true) {
  let timer: ReturnType<typeof setTimeout> | undefined
  let lastRun = Date.now()
  let running = false
  let disposed = false

  const clear = () => {
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
  }
  const schedule = () => {
    clear()
    if (disposed || running || document.hidden || !enabled()) return
    timer = setTimeout(run, Math.max(0, lastRun + intervalMs - Date.now()))
  }
  const run = async () => {
    running = true
    lastRun = Date.now()
    try {
      await task()
    } catch {
      // A failed background refresh keeps the last data; the next tick retries.
    } finally {
      running = false
      schedule()
    }
  }

  onMounted(() => {
    lastRun = Date.now()
    document.addEventListener('visibilitychange', schedule)
    schedule()
  })
  watch(enabled, (active) => {
    if (active) lastRun = Date.now()
    schedule()
  })
  onBeforeUnmount(() => {
    disposed = true
    clear()
    document.removeEventListener('visibilitychange', schedule)
  })
}
