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
  results: [{ case_name: 'case-1', message: 'hidden result details' }]
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
  it('renders the shared four-card source view without message or result details', () => {
    const wrapper = mount(SubmissionDetailDialog, {
      props: { modelValue: true, detail },
      global: { stubs: globalStubs }
    })

    expect(wrapper.get('.dialog-stub').attributes('data-title')).toBe('提交代码')
    expect(wrapper.findAll('.submission-detail-summary article')).toHaveLength(4)
    expect(wrapper.text()).toContain('张三')
    expect(wrapper.text()).not.toContain('2026001')
    expect(wrapper.text()).toContain('Python 3')
    expect(wrapper.text()).toContain('Unaccepted')
    expect(wrapper.get('svg').attributes('aria-label')).toBe('未通过')
    expect(wrapper.get('.submission-source-code').text()).toContain('print("hello")')
    expect(wrapper.text()).not.toContain('some test cases failed')
    expect(wrapper.text()).not.toContain('hidden result details')
  })
})
