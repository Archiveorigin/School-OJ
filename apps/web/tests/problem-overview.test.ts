import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ProblemOverview from '../src/components/ProblemOverview.vue'

describe('ProblemOverview', () => {
  it('marks accepted and attempted rows and emits the selected problem', async () => {
    const wrapper = mount(ProblemOverview, {
      props: {
        activeProblemId: 2,
        items: [
          { problem: { id: 1, title: 'Accepted', statement: '', time_limit_ms: 1000, memory_limit_mb: 256, output_limit_kb: 1024 }, label: 'A', score: 100, submission_status: 'accepted' },
          { problem: { id: 2, title: 'Attempted', statement: '', time_limit_ms: 1000, memory_limit_mb: 256, output_limit_kb: 1024 }, label: 'B', score: 100, submission_status: 'wrong_answer' }
        ]
      },
      global: { stubs: { ElTag: true, ElEmpty: true } }
    })
    const rows = wrapper.findAll('tbody tr')
    expect(rows[0].classes()).toContain('accepted')
    expect(rows[1].classes()).toContain('attempted')
    expect(rows[1].classes()).toContain('active')
    await rows[0].trigger('click')
    expect(wrapper.emitted('select')?.[0]).toEqual([1])
  })
})
