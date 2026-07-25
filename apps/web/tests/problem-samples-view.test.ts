import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ProblemSamplesView from '../src/components/ProblemSamplesView.vue'

describe('ProblemSamplesView', () => {
  it('renders the sample name once and stacks one input above one output', () => {
    const wrapper = mount(ProblemSamplesView, {
      props: {
        samples: [{ index: 1, name: '样例 1', input: '1 2', output: '3' }]
      },
      global: {
        stubs: {
          ElButton: { template: '<button><slot /></button>' }
        }
      }
    })

    expect(wrapper.text().match(/样例 1/g)).toHaveLength(1)
    expect(wrapper.findAll('.sample-block')).toHaveLength(2)
    expect(wrapper.find('.sample-pair').classes()).toContain('sample-pair')
  })
})
