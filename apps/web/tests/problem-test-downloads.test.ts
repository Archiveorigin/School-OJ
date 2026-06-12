import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ProblemTestDownloads from '../src/components/ProblemTestDownloads.vue'
import { client } from '../src/api/client'

vi.mock('element-plus', () => ({
  ElMessage: {
    error: vi.fn()
  }
}))

vi.mock('../src/api/client', () => ({
  client: {
    get: vi.fn()
  }
}))

describe('ProblemTestDownloads', () => {
  it('loads and renders available test files', async () => {
    vi.mocked(client.get).mockResolvedValueOnce({
      data: {
        tests: [{ name: 'case-01', input: 'tests/01.in', output: 'tests/01.out', weight: 100 }]
      }
    })

    const wrapper = mount(ProblemTestDownloads, {
      props: { problemId: 7, problemCode: 'T007' },
      global: {
        stubs: {
          ElButton: { template: '<button><slot /></button>' },
          ElTable: { props: ['data'], template: '<div><div v-for="row in data" class="test-row">{{ row.name }} {{ row.input }} {{ row.output }}</div></div>' },
          ElTableColumn: true
        }
      }
    })
    await flushPromises()

    expect(client.get).toHaveBeenCalledWith('/problems/7/tests')
    expect(wrapper.text()).toContain('case-01')
    expect(wrapper.text()).toContain('tests/01.in')
    expect(wrapper.text()).toContain('tests/01.out')
  })
})
