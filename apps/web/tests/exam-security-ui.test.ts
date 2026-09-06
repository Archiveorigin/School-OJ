import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { ExamFocusState, isDevtoolsShortcut, ExamActivityQueue } from '../src/features/exams/activity'
import { sourceLineNumbers, hasEvaluatedResources } from '../src/features/submissions/presentation'
import { legacyExamPath, examProblemItems, problemLabel } from '../src/features/events/workspace'
import { parseTeamJoinSlug } from '../src/features/teams/join'
import EventWorkspaceHeader from '../src/components/EventWorkspaceHeader.vue'
import { client, generatedApi } from '../src/api/client'
import { createPinia, setActivePinia } from 'pinia'
import { useExamLockStore } from '../src/stores/examLock'
import { adaptExamRanking, adaptTeamContestRanking } from '../src/features/leaderboard/adapters'
vi.mock('../src/api/client', () => ({ client: { get: vi.fn() }, generatedApi: { recordExamActivity: vi.fn() }, apiOperations: { recordExamActivity: { path: '/exams/{id}/activity' } }, apiBase: '/api' }))
vi.mock('../src/api/http', () => ({ getActiveToken: () => 'test-session' }))

beforeEach(() => { localStorage.clear(); vi.mocked(generatedApi.recordExamActivity).mockReset(); vi.mocked(client.get).mockReset(); setActivePinia(createPinia()) })
describe('exam environment', () => {
  it('deduplicates blur/visibility and records a single matching return', () => {
    const events: string[] = [], state = new ExamFocusState(event => events.push(event))
    state.update(true, false); state.update(false, false); state.update(false, true); state.update(true, true); state.update(true, true)
    expect(events).toEqual(['window_leave', 'window_return'])
  })
  it('intercepts only developer shortcuts, not editing copy or ordinary F keys', () => {
    const key = (key: string, ctrlKey = false, shiftKey = false, metaKey = false, altKey = false) => ({ key, ctrlKey, shiftKey, metaKey, altKey })
    expect(isDevtoolsShortcut(key('F12'))).toBe(true)
    expect(isDevtoolsShortcut(key('I', true, true))).toBe(true)
    expect(isDevtoolsShortcut(key('j', false, false, true, true))).toBe(true)
    expect(isDevtoolsShortcut(key('c', true))).toBe(false)
    expect(isDevtoolsShortcut(key('F5'))).toBe(false)
  })
  it('persists failed requests and retries with stable event IDs', async () => {
    const queue = new ExamActivityQueue(4, 'test-queue')
    queue.push('window_leave'); vi.mocked(generatedApi.recordExamActivity).mockRejectedValueOnce(new Error('offline'))
    await queue.flush()
    const first = JSON.parse(localStorage.getItem('test-queue') || '[]')
    expect(first).toHaveLength(1)
    vi.mocked(generatedApi.recordExamActivity).mockResolvedValueOnce({ data: { recorded: true } } as any)
    await new ExamActivityQueue(4, 'test-queue').flush()
    expect(vi.mocked(generatedApi.recordExamActivity).mock.calls[1][0]).toEqual({ path: { id: 4 }, body: { events: first } })
    expect(localStorage.getItem('test-queue')).toBeNull()
  })
  it('bounds storage and upload batches', async () => {
    const queue = new ExamActivityQueue(4, 'test-queue')
    for (let i = 0; i < 205; i++) queue.push('window_leave')
    expect(JSON.parse(localStorage.getItem('test-queue') || '[]')).toHaveLength(200)
    vi.mocked(generatedApi.recordExamActivity).mockResolvedValueOnce({} as any)
    await queue.flush()
    expect(vi.mocked(generatedApi.recordExamActivity).mock.calls[0][0]?.body?.events).toHaveLength(30)
    expect(JSON.parse(localStorage.getItem('test-queue') || '[]')).toHaveLength(170)
  })
})
describe('shared event and submission presentation', () => {
  it('keeps frozen state explicit for exam and contest mobile headers', () => {
    expect(adaptExamRanking({ frozen: true, exam: { scoring_rule: 'ioi' } }).frozen).toBe(true)
    expect(adaptTeamContestRanking({ frozen: true }).frozen).toBe(true)
  })
  it('keeps OI navigation free of a public leaderboard', () => {
    const wrapper = mount(EventWorkspaceHeader, { props: { title: '课程考试', kind: '考试', rule: 'oi', status: '进行中', showRanking: true, base: '/exams/1' }, global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } } })
    expect(wrapper.text()).toContain('考试说明'); expect(wrapper.text()).not.toContain('排行榜')
  })
  it('adapts exam scores and legacy links without treating every exam problem as 100 points', () => {
    const items = examProblemItems([{ problem_id: 1, problem: { id: 1 } as any, score: 35 }], [{ problem_id: 1, status: 'recorded' } as any])
    expect(items[0]).toMatchObject({ score: 35, label: 'A', submission_status: 'recorded' })
    expect(problemLabel(26)).toBe('AA')
    expect(legacyExamPath('3', 'overview', {})).toBe('/exams/3/description')
    expect(legacyExamPath('3', 'ranking', {})).toBe('/exams/3/scoreboard')
    expect(legacyExamPath('3', 'records', { label: 'B' })).toBe('/exams/3/problems/B')
  })
  it('numbers blank lines and different line endings while keeping results hidden', () => {
    expect(sourceLineNumbers('a\r\n\r\n<code>\n')).toBe('1\n2\n3\n4')
    expect(hasEvaluatedResources('recorded')).toBe(false)
    expect(hasEvaluatedResources('pending_review')).toBe(false)
  })
  it('accepts team slugs and links but never navigates to an arbitrary external URL', () => {
    expect(parseTeamJoinSlug(' HTTPS://example.com/teams/my-team/join ')).toBe('my-team')
    expect(parseTeamJoinSlug('my-team')).toBe('my-team')
    expect(parseTeamJoinSlug('https://example.com/admin')).toBe('')
    expect(parseTeamJoinSlug('../private')).toBe('')
  })
})

describe('server-authoritative exam state', () => {
  it('restores a lock from the server without relying on local storage', async () => {
    const store = useExamLockStore()
    vi.mocked(client.get).mockResolvedValueOnce({ data: { active: true, exam: { id: 4, title: '考试' } } })
    await store.syncActiveExam()
    expect(store.locked).toBe(true); expect(store.examId).toBe(4); expect(store.synced).toBe(true)
  })
  it('retains the last lock on failed status checks', async () => {
    const store = useExamLockStore(); store.lock(4)
    vi.mocked(client.get).mockRejectedValueOnce(new Error('service unavailable'))
    await expect(store.syncActiveExam()).rejects.toThrow()
    expect(store.locked).toBe(true); expect(store.synced).toBe(false)
  })
  it('removes stale client locks only when the server confirms completion', async () => {
    const store = useExamLockStore(); store.lock(4)
    vi.mocked(client.get).mockResolvedValueOnce({ data: { active: false } })
    await store.syncActiveExam()
    expect(store.locked).toBe(false); expect(localStorage.getItem('school-oj-active-exam')).toBeNull()
  })
})
