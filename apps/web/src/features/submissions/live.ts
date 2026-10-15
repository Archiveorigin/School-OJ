import { onBeforeUnmount } from 'vue'
import { openEventStream, type AuthenticatedEventSource, type Submission } from '../../api/client'

// Each watcher owns its stream. A late terminal event cannot close a newer
// submission's stream, and leaving any submission page releases every watcher.
export function useSubmissionStreams() {
  const streams = new Set<AuthenticatedEventSource>()
  const closeAll = () => { for (const stream of streams) stream.close(); streams.clear() }
  onBeforeUnmount(closeAll)
  function watch(id: number, update: (state: Submission) => void, complete?: () => void | Promise<void>) {
    const stream = openEventStream(`/submissions/${id}/events`)
    streams.add(stream)
    let completed = false
    stream.addEventListener('status', (event: MessageEvent<string>) => {
      if (completed) return
      let state: Submission
      try { state = JSON.parse(event.data) } catch { return }
      if (!state || typeof state.status !== 'string') return
      update(state)
      if (!['queued', 'running'].includes(state.status)) {
        completed = true
        stream.close()
        streams.delete(stream)
        if (complete) void Promise.resolve().then(complete).catch(() => { /* page refresh handles its own errors */ })
      }
    })
    return stream
  }
  return { watch, closeAll }
}
