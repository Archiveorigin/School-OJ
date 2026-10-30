import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { shortVerdicts } from '../src/features/submissions/presentation'
import SubmissionStatusMark from '../src/components/SubmissionStatusMark.vue'

describe('SubmissionStatusMark', () => {
  it('renders an accepted submission as an accessible green inline svg', () => {
    const wrapper = mount(SubmissionStatusMark, {
      props: { status: 'accepted' },
    })
    const svg = wrapper.get('svg')

    expect(svg.attributes('role')).toBe('img')
    expect(svg.attributes('aria-label')).toBe('通过')
    expect(wrapper.text()).toContain('AC')
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

    expect(wrapper.get('svg').attributes('aria-label')).not.toBe('评测中')
    expect(wrapper.get('text').text()).toBe(shortVerdicts[status])
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

it.each([['accepted','AC'],['wrong_answer','WA'],['compile_error','CE'],['time_limit','TLE'],['memory_limit','MLE'],['runtime_error','RE']])('shares compact SVG result %s', (status,code)=>{
 const wrapper=mount(SubmissionStatusMark,{props:{status,compact:true}})
 expect(wrapper.get('text').text()).toBe(code)
 expect(wrapper.get('title').text().length).toBeGreaterThan(0)
})
