import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SubmissionDetailDialog from '../src/components/SubmissionDetailDialog.vue'

const detail = {
  submission: {
    id: 18,
    user_id: 7,
    problem_id: 3,
    language: 'python',
    source_code: 'print("hello")',
    status: 'runtime_error',
    score: 0,
    time_ms: 32,
    memory_kb: 2048,
    message: 'some test cases failed',
    user_name: '张三',
    student_no: '2026001',
    created_at: '2026-08-03T00:40:37Z',
    updated_at: '2026-08-03T00:40:38Z'
  },
  results: [{ case_name: 'case-1', message: 'hidden result details' }],
  case_summary: { passed: 0, total: 10 }
}

const globalStubs = {
  ElDialog: {
    props: ['modelValue', 'title'],
    template: '<section class="dialog-stub" :data-title="title"><slot /></section>'
  },
  ElButton: {
    template: '<button><slot /></button>'
  }
}

describe('SubmissionDetailDialog', () => {
  it('renders inline metadata, short verdict, version case total and white code viewer', () => {
    const wrapper = mount(SubmissionDetailDialog, {
      props: { modelValue: true, detail },
      global: { stubs: globalStubs }
    })

    expect(wrapper.get('.dialog-stub').attributes('data-title')).toBe('提交详情')
    expect(wrapper.findAll('.submission-detail-summary article')).toHaveLength(0)
    expect(wrapper.text()).toContain('运行时间：32ms')
    expect(wrapper.text()).toContain('占用内存：2048KB')
    expect(wrapper.text()).toContain('用例通过情况：0/10')
    expect(wrapper.text()).not.toContain('2026001')
    expect(wrapper.text()).toContain('Python 3')
    expect(wrapper.text()).toContain('RE')
    expect(wrapper.get('svg').attributes('aria-label')).toBe('未通过')
    expect(wrapper.get('.source-code').text()).toContain('print("hello")')
    expect(wrapper.get('.source-lines').text()).toBe('1')
    expect(wrapper.text()).not.toContain('some test cases failed')
    expect(wrapper.text()).not.toContain('hidden result details')
  })
  it('never renders hidden OI/frozen resources or case counts', () => {
    const wrapper = mount(SubmissionDetailDialog, { props: { modelValue: true, detail: { ...detail, submission: { ...detail.submission, status: 'recorded', time_ms: 999, memory_kb: 888 } } }, global: { stubs: globalStubs } })
    expect(wrapper.text()).toContain('已记录')
    expect(wrapper.text()).toContain('用例通过情况：—')
    expect(wrapper.text()).not.toContain('999')
    expect(wrapper.text()).not.toContain('888')
  })
})
