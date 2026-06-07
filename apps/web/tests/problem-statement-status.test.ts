import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ProblemStatementView from '../src/components/ProblemStatementView.vue'

const problem = {
  id: 1,
  title: 'A + B Problem',
  statement: '',
  time_limit_ms: 1000,
  memory_limit_kb: 262144,
  tags: ''
}

const globalStubs = {
  MarkdownRenderer: true,
  ElButton: true,
  ElTag: {
    template: '<span class="el-tag-stub"><slot /></span>'
  }
}

describe('ProblemStatementView status icon', () => {
  it('renders accepted status as inline svg', () => {
    const wrapper = mount(ProblemStatementView, {
      props: {
        problem,
        statusImage: 'ac'
      },
      global: {
        stubs: globalStubs
      }
    })

    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('svg.status-icon').exists()).toBe(true)
    expect(wrapper.find('[role="img"]').attributes('aria-label')).toBe('通过')
    expect(wrapper.text()).toContain('Accepted')
    expect(wrapper.html()).toContain('#22c55e')
  })

  it('renders unaccepted status as inline svg', () => {
    const wrapper = mount(ProblemStatementView, {
      props: {
        problem,
        statusImage: 'uac'
      },
      global: {
        stubs: globalStubs
      }
    })

    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('svg.status-icon').exists()).toBe(true)
    expect(wrapper.find('[role="img"]').attributes('aria-label')).toBe('未通过')
    expect(wrapper.text()).toContain('Unaccepted')
    expect(wrapper.html()).toContain('#ef4444')
  })

  it('keeps the original unsubmitted fallback without an svg', () => {
    const wrapper = mount(ProblemStatementView, {
      props: {
        problem
      },
      global: {
        stubs: globalStubs
      }
    })

    expect(wrapper.find('svg.status-icon').exists()).toBe(false)
    expect(wrapper.text()).toContain('未提交')
  })
})
