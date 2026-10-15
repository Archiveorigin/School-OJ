import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { expect, it, vi } from 'vitest'
import { useSubmissionStreams } from './live'

const mocks = vi.hoisted(() => ({ streams: [] as any[] }))
vi.mock('../../api/client', () => ({ openEventStream: () => {
  const stream = { close: vi.fn(), listener: undefined as any, addEventListener: (_: string, fn: any) => { stream.listener = fn } }
  mocks.streams.push(stream)
  return stream
} }))

it('isolates terminal events and closes all pending streams on unmount', async () => {
  let watchers!: ReturnType<typeof useSubmissionStreams>
  const wrapper = mount(defineComponent({ setup() { watchers = useSubmissionStreams(); return () => null } }))
  const complete = vi.fn()
  watchers.watch(1, vi.fn(), complete)
  watchers.watch(2, vi.fn())
  const [first, second] = mocks.streams
  first.listener(new MessageEvent('status', { data: '{"status":"accepted"}' }))
  first.listener(new MessageEvent('status', { data: '{"status":"accepted"}' }))
  await Promise.resolve()
  expect(complete).toHaveBeenCalledTimes(1)
  expect(first.close).toHaveBeenCalledTimes(1)
  expect(second.close).not.toHaveBeenCalled()
  wrapper.unmount()
  expect(second.close).toHaveBeenCalledTimes(1)
})
