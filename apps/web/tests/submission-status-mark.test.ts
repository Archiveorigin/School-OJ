import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SubmissionStatusMark from '../src/components/SubmissionStatusMark.vue'

describe('SubmissionStatusMark', () => {
  it('renders an accepted submission as an accessible green inline svg', () => {
    const wrapper = mount(SubmissionStatusMark, {
      props: { status: 'accepted' },
    })
    const svg = wrapper.get('svg')

    expect(svg.attributes('role')).toBe('img')
    expect(svg.attributes('aria-label')).toBe('通过')
    expect(wrapper.text()).toContain('Accepted')
    expect(wrapper.get('rect').attributes('fill')).toBe('#22c55e')
  })

  it.each([
    'wrong_answer',
    'compile_error',
    'runtime_error',
    'time_limit',
    'memory_limit',
    'output_limit',
    'system_error',
  ])('renders the terminal failure %s as Unaccepted', (status) => {
    const wrapper = mount(SubmissionStatusMark, { props: { status } })

    expect(wrapper.get('svg').attributes('aria-label')).toBe('未通过')
    expect(wrapper.text()).toContain('Unaccepted')
    expect(wrapper.get('rect').attributes('fill')).toBe('#ef4444')
  })

  it('does not render a running submission as a failure', () => {
    const wrapper = mount(SubmissionStatusMark, {
      props: { status: 'running' },
    })

    expect(wrapper.get('svg').attributes('aria-label')).toBe('评测中')
    expect(wrapper.text()).toContain('评测中')
    expect(wrapper.text()).not.toContain('Unaccepted')
    expect(wrapper.get('rect').attributes('fill')).toBe('#3b82f6')
  })

  it('renders a manually graded submission as its own non-failure state', () => {
    const wrapper = mount(SubmissionStatusMark, {
      props: { status: 'manual_graded' },
    })

    expect(wrapper.get('svg').attributes('aria-label')).toBe('已评分')
    expect(wrapper.text()).toContain('已评分')
    expect(wrapper.text()).not.toContain('Unaccepted')
    expect(wrapper.get('rect').attributes('fill')).toBe('#d97706')
  })
})
