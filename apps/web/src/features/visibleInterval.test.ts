import { mount } from '@vue/test-utils'
import { defineComponent, nextTick, ref } from 'vue'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { useVisibleInterval } from './visibleInterval'

let hidden = false
beforeEach(() => {
  vi.useFakeTimers()
  hidden = false
  Object.defineProperty(document, 'hidden', { configurable: true, get: () => hidden })
})
afterEach(() => vi.useRealTimers())

function setHidden(value: boolean) {
  hidden = value
  document.dispatchEvent(new Event('visibilitychange'))
}
function mountWith(task: () => unknown, enabled?: () => boolean) {
  return mount(defineComponent({ setup() { useVisibleInterval(task, 1000, enabled); return () => null } }))
}

it('repeats only while visible and catches up once on return', async () => {
  const task = vi.fn()
  const wrapper = mountWith(task)
  await vi.advanceTimersByTimeAsync(2500)
  expect(task).toHaveBeenCalledTimes(2)
  setHidden(true)
  await vi.advanceTimersByTimeAsync(60000)
  expect(task).toHaveBeenCalledTimes(2)
  setHidden(false)
  await vi.advanceTimersByTimeAsync(0)
  expect(task).toHaveBeenCalledTimes(3)
  await vi.advanceTimersByTimeAsync(999)
  expect(task).toHaveBeenCalledTimes(3)
  wrapper.unmount()
  await vi.advanceTimersByTimeAsync(5000)
  expect(task).toHaveBeenCalledTimes(3)
})

it('does not run early when a tab returns before the interval elapsed', async () => {
  const task = vi.fn()
  const wrapper = mountWith(task)
  await vi.advanceTimersByTimeAsync(400)
  setHidden(true)
  await vi.advanceTimersByTimeAsync(100)
  setHidden(false)
  await vi.advanceTimersByTimeAsync(0)
  expect(task).not.toHaveBeenCalled()
  await vi.advanceTimersByTimeAsync(500)
  expect(task).toHaveBeenCalledTimes(1)
  wrapper.unmount()
})

it('follows the enabled condition without firing on the transition', async () => {
  const task = vi.fn()
  const active = ref(false)
  const wrapper = mountWith(task, () => active.value)
  await vi.advanceTimersByTimeAsync(5000)
  expect(task).not.toHaveBeenCalled()
  active.value = true
  await nextTick()
  await vi.advanceTimersByTimeAsync(999)
  expect(task).not.toHaveBeenCalled()
  await vi.advanceTimersByTimeAsync(1)
  expect(task).toHaveBeenCalledTimes(1)
  active.value = false
  await nextTick()
  await vi.advanceTimersByTimeAsync(5000)
  expect(task).toHaveBeenCalledTimes(1)
  wrapper.unmount()
})

it('never overlaps a slow task and survives a failed one', async () => {
  let finish!: () => void
  const task = vi.fn(() => new Promise<void>((resolve) => { finish = resolve }))
  const wrapper = mountWith(task)
  await vi.advanceTimersByTimeAsync(5000)
  expect(task).toHaveBeenCalledTimes(1)
  finish()
  await vi.advanceTimersByTimeAsync(0)
  expect(task).toHaveBeenCalledTimes(2)
  wrapper.unmount()

  const failing = vi.fn(async () => { throw new Error('offline') })
  const second = mountWith(failing)
  await vi.advanceTimersByTimeAsync(2100)
  expect(failing).toHaveBeenCalledTimes(2)
  second.unmount()
})
