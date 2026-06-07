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

describe('ProblemStatementView status icon', () => {
  it('renders accepted status as inline svg', () => {
    const wrapper = mount(ProblemStatementView, {
      props: {
        problem,
        statusImage: 'ac'
      },
      global: {
        stubs: {
          MarkdownRenderer: true,
          ElButton: true,
          ElTag: true
        }
      }
    })

    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('svg.status-icon').exists()).toBe(true)
    expect(wrapper.find('[role="img"]').attributes('aria-label')).toBe('通过')
    expect(wrapper.html()).toContain('#22c55e')
  })

  it('renders unaccepted status as inline svg', () => {
    const wrapper = mount(ProblemStatementView, {
      props: {
        problem,
        statusImage: 'uac'
      },
      global: {
        stubs: {
          MarkdownRenderer: true,
          ElButton: true,
          ElTag: true
        }
      }
    })

    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('svg.status-icon').exists()).toBe(true)
    expect(wrapper.find('[role="img"]').attributes('aria-label')).toBe('未通过')
    expect(wrapper.html()).toContain('#ef4444')
  })
})
