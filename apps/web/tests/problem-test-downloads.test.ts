import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProblemTestDownloads from '../src/components/ProblemTestDownloads.vue'
import { generatedApi } from '../src/api/client'

vi.mock('element-plus', () => ({
  ElMessage: {
    error: vi.fn(),
    success: vi.fn()
  }
}))

vi.mock('../src/api/client', () => ({
  generatedApi: {
    listProblemTests: vi.fn(), downloadProblemTestFile: vi.fn()
  }
}))

describe('ProblemTestDownloads', () => {
  beforeEach(() => vi.clearAllMocks())
  it('opens test data in a dialog and loads input/output content', async () => {
    vi.mocked(generatedApi.listProblemTests).mockResolvedValue({
          data: {
            tests: [{ name: 'case-01', input: 'tests/01.in', output: 'tests/01.out', weight: 100 }]
          }
        } as any)
    vi.mocked(generatedApi.downloadProblemTestFile).mockImplementation(async (options) => {
      if (options?.path?.file_path === 'tests/01.in') return { data: '1 2\n' } as any
      if (options?.path?.file_path === 'tests/01.out') return { data: '3\n' } as any
      throw new Error('unexpected test file')
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

    expect(generatedApi.listProblemTests).not.toHaveBeenCalled()
    expect(wrapper.find('.dialog').exists()).toBe(false)

    const trigger = wrapper.findAll('button').find((button) => button.text().includes('测试数据'))
    await trigger?.trigger('click')
    await flushPromises()

    expect(generatedApi.listProblemTests).toHaveBeenCalledWith(expect.objectContaining({ path: { id: 7 } }))
    expect(wrapper.text()).toContain('case-01')
    expect(wrapper.text()).toContain('tests/01.in')
    expect(wrapper.text()).toContain('tests/01.out')
    expect(wrapper.text()).toContain('1 2')
    expect(wrapper.text()).toContain('3')
    wrapper.unmount()
  })

  it('ignores stale metadata after a question switch and aborts on unmount', async () => {
    let resolveOld!: (value: any) => void
    vi.mocked(generatedApi.listProblemTests).mockImplementation((options) => options?.path?.id === 1
      ? new Promise(resolve => { resolveOld = resolve })
      : Promise.resolve({ data: { tests: [] } } as any))
    const wrapper = mount(ProblemTestDownloads, { props: { problemId: 1 }, global: { stubs: {
      ElButton: { template: '<button><slot /></button>' },
      ElDialog: { props: ['modelValue'], template: '<div v-if="modelValue"><slot /></div>' },
      ElSkeleton: true, ElEmpty: true,
    } } })
    await wrapper.get('button').trigger('click')
    const firstSignal = vi.mocked(generatedApi.listProblemTests).mock.calls[0][0]!.signal!
    await wrapper.setProps({ problemId: 2 })
    await flushPromises()
    expect(firstSignal.aborted).toBe(true)
    resolveOld({ data: { tests: [{ name: 'old', input: 'old.in', output: 'old.out', weight: 1 }] } })
    await flushPromises()
    expect(generatedApi.downloadProblemTestFile).not.toHaveBeenCalled()
    expect(wrapper.text()).not.toContain('old.in')
    const lastSignal = vi.mocked(generatedApi.listProblemTests).mock.calls.at(-1)![0]!.signal!
    wrapper.unmount()
    expect(lastSignal.aborted).toBe(true)
  })
})
