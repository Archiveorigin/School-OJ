import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SubmissionComposer from '../src/components/SubmissionComposer.vue'
import { loadSubmissionDraft } from '../src/features/submissions/drafts'

const scope = { userId: 1, resourceType: 'problem' as const, resourceId: 2, problemId: 2 }
const stubs = { CodeEditor: { name: 'CodeEditor', props: ['modelValue'], emits: ['update:modelValue'], template: '<div />' }, VerdictTag: true, ElSelect: true, ElOption: true, ElButton: { template: '<button><slot /></button>' } }
const create = () => mount(SubmissionComposer, { props: { source: '', language: 'cpp', draftContext: scope }, global: { stubs } })
beforeEach(() => { localStorage.clear(); vi.useFakeTimers() })
afterEach(() => { vi.useRealTimers(); vi.restoreAllMocks() })

describe('submission drafts on the typing path', () => {
  it('emits every edit immediately but coalesces storage writes', async () => {
    const wrapper = create()
    const save = vi.spyOn(Storage.prototype, 'setItem')
    for (const value of ['a', 'ab', 'abc']) wrapper.getComponent({ name: 'CodeEditor' }).vm.$emit('update:modelValue', value)
    expect(wrapper.emitted('update:source')).toEqual([['a'], ['ab'], ['abc']])
    expect(save).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(300)
    expect(save).toHaveBeenCalledTimes(1)
    expect(loadSubmissionDraft(scope, 'cpp')).toBe('abc')
    wrapper.unmount()
  })
  it('flushes the captured old scope on question switch and page exit', async () => {
    const wrapper = create()
    wrapper.getComponent({ name: 'CodeEditor' }).vm.$emit('update:modelValue', 'old question')
    const next = { ...scope, problemId: 3 }
    await wrapper.setProps({ draftContext: next })
    expect(loadSubmissionDraft(scope, 'cpp')).toBe('old question')
    wrapper.getComponent({ name: 'CodeEditor' }).vm.$emit('update:modelValue', 'new question')
    window.dispatchEvent(new Event('pagehide'))
    expect(loadSubmissionDraft(next, 'cpp')).toBe('new question')
    wrapper.unmount()
  })
  it('does not resurrect a cleared draft and tolerates storage quota errors', async () => {
    const wrapper = create()
    wrapper.getComponent({ name: 'CodeEditor' }).vm.$emit('update:modelValue', 'submitted')
    wrapper.vm.clearDraft()
    await vi.advanceTimersByTimeAsync(400)
    expect(loadSubmissionDraft(scope, 'cpp')).toBeNull()
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('quota') })
    wrapper.getComponent({ name: 'CodeEditor' }).vm.$emit('update:modelValue', 'still editable')
    await expect(vi.advanceTimersByTimeAsync(400)).resolves.not.toThrow()
    expect(wrapper.emitted('update:source')?.at(-1)).toEqual(['still editable'])
    wrapper.unmount()
  })
})

 it('blocks submission events while disabled or already submitting', async () => {
  const wrapper=create()
  await wrapper.setProps({submitting:true})
  await wrapper.findAll('button').at(-1)!.trigger('click')
  expect(wrapper.emitted('submit')).toBeUndefined()
  await wrapper.setProps({submitting:false,disabled:true})
  await wrapper.findAll('button').at(-1)!.trigger('click')
  expect(wrapper.emitted('submit')).toBeUndefined()
  wrapper.unmount()
 })
