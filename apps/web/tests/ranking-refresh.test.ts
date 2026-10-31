import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ExamRanking from '../src/views/exam/ExamRanking.vue'
import EventRankingReview from '../src/components/events/EventRankingReview.vue'
import { generatedApi } from '../src/api/client'
import { saveDownload } from '../src/api/modules/events'

vi.mock('vue-router', () => ({ useRoute: () => ({ params: { id: '7' } }) }))
vi.mock('../src/api/client', () => ({ generatedApi: { examRanking: vi.fn(), getEventRanking: vi.fn(), exportEventRanking: vi.fn() } }))
vi.mock('../src/components/LeaderboardBoard.vue', () => ({ default: { name: 'LeaderboardBoard', emits: ['refresh'], template: '<button class="manual-refresh" @click="$emit(\'refresh\')">刷新榜单</button>' } }))
vi.mock('../src/api/modules/events', async importOriginal => ({ ...await importOriginal<any>(), saveDownload: vi.fn() }))

beforeEach(() => { vi.useFakeTimers();setActivePinia(createPinia()) })
afterEach(() => { vi.useRealTimers(); vi.clearAllMocks() })
const snapshot = (judging: number) => ({
  title: '截止后的成绩', lifecycle: 'closed', scoring_rule: 'acm', generated_at: new Date().toISOString(),
  completion_checks: { judging, ungraded: 0 }, rows: [], problems: [],
})

describe('ranking refresh across the deadline', () => {
  it('fetches on entry and manual refresh without a periodic timer', async () => {
    vi.mocked(generatedApi.examRanking).mockResolvedValue({
      data: { lifecycle: 'closed', exam: { ends_at: '2000-01-01T00:00:00Z' }, rows: [] },
    } as any)
    const wrapper = mount(ExamRanking)
    await flushPromises()
    expect(generatedApi.examRanking).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()
    expect(generatedApi.examRanking).toHaveBeenCalledTimes(1)
    await wrapper.get('.manual-refresh').trigger('click')
    await flushPromises()
    expect(generatedApi.examRanking).toHaveBeenCalledTimes(2)
    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(15000)
    expect(generatedApi.examRanking).toHaveBeenCalledTimes(2)
  })

  it('enables actual export only after pending judging settles and keeps the full server export', async () => {
    vi.mocked(generatedApi.getEventRanking)
      .mockResolvedValueOnce({ data: snapshot(1) } as any)
      .mockResolvedValue({ data: snapshot(0) } as any)
    const blob = new Blob(['standings'])
    vi.mocked(generatedApi.exportEventRanking).mockResolvedValue({ data: blob } as any)
    const wrapper = mount(EventRankingReview, {
      props: { kind: 'contest', eventId: 7 },
      global: { stubs: { ElButton: { props: ['disabled'], template: '<button :disabled="disabled"><slot /></button>' } } },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('待评测 1 份')
    expect(wrapper.findAll('button')[0].attributes('disabled')).toBeDefined()
    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()
    expect(generatedApi.getEventRanking).toHaveBeenCalledTimes(1)
    await wrapper.get('.manual-refresh').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('当前为最终排名')
    expect(wrapper.findAll('button')[0].attributes('disabled')).toBeUndefined()
    await wrapper.findAll('button')[0].trigger('click')
    await flushPromises()
    expect(generatedApi.exportEventRanking).toHaveBeenCalledWith({
      path: { kind: 'contest', event_id: 7 }, query: { format: 'xlsx' }, responseType: 'blob',
    })
    expect(saveDownload).toHaveBeenCalledWith(blob, '截止后的成绩.xlsx')
    wrapper.unmount()
  })

  it('keeps export disabled while a judge system failure needs attention', async () => {
    vi.mocked(generatedApi.getEventRanking).mockResolvedValue({
      data: { ...snapshot(0), completion_checks: { judging: 0, ungraded: 0, failed: 1 } },
    } as any)
    const wrapper = mount(EventRankingReview, {
      props: { kind: 'exam', eventId: 7 },
      global: { stubs: { ElButton: { props: ['disabled'], template: '<button :disabled="disabled"><slot /></button>' } } },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('判题异常 1 份')
    expect(wrapper.findAll('button')[0].attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
})
