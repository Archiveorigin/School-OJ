import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ProblemTestDownloads from '../src/components/ProblemTestDownloads.vue'
import { client } from '../src/api/client'

vi.mock('element-plus', () => ({
  ElMessage: {
    error: vi.fn(),
    success: vi.fn()
  }
}))

vi.mock('../src/api/client', () => ({
  client: {
    get: vi.fn()
  }
}))

describe('ProblemTestDownloads', () => {
  it('opens test data in a dialog and loads input/output content', async () => {
    vi.mocked(client.get).mockImplementation(async (url) => {
      if (url === '/problems/7/tests') {
        return {
          data: {
            tests: [{ name: 'case-01', input: 'tests/01.in', output: 'tests/01.out', weight: 100 }]
          }
        } as any
      }
      if (String(url).endsWith('/tests/01.in')) return { data: '1 2\n' } as any
      if (String(url).endsWith('/tests/01.out')) return { data: '3\n' } as any
      throw new Error(`unexpected request: ${url}`)
    })

    const wrapper = mount(ProblemTestDownloads, {
      props: { problemId: 7, problemCode: 'T007' },
      global: {
        stubs: {
          ElButton: { template: '<button @click="$emit(\'click\')"><slot /></button>' },
          ElDialog: { props: ['modelValue'], template: '<div v-if="modelValue" class="dialog"><slot /><slot name="footer" /></div>' },
          ElSkeleton: true,
          ElTag: { template: '<span><slot /></span>' },
          ElEmpty: true
        }
      }
    })
    await flushPromises()

    expect(client.get).toHaveBeenCalledWith('/problems/7/tests')
    expect(wrapper.find('.dialog').exists()).toBe(false)

    const trigger = wrapper.findAll('button').find((button) => button.text().includes('测试数据'))
    await trigger?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('case-01')
    expect(wrapper.text()).toContain('tests/01.in')
    expect(wrapper.text()).toContain('tests/01.out')
    expect(wrapper.text()).toContain('1 2')
    expect(wrapper.text()).toContain('3')
  })
})
