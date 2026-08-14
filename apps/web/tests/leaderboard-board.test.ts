import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import LeaderboardBoard from '../src/components/LeaderboardBoard.vue'
import { sortLeaderboardRows } from '../src/features/leaderboard/sorting'
import type { LeaderboardData, LeaderboardRow } from '../src/features/leaderboard/types'

function row(overrides: Partial<LeaderboardRow> = {}): LeaderboardRow {
  return {
    id: 1,
    rank: 1,
    name: '章佳荣',
    studentNo: 'STUDENT-NUMBER-MUST-NOT-RENDER',
    meta: 'SUBMISSION-METADATA-MUST-NOT-RENDER',
    solved: 1,
    metric: 75,
    metricDisplay: '75',
    submissions: 4,
    results: {
      '1': { status: 'accepted', attempts: 2, primary: '2', secondary: "35'", firstBlood: true }
    },
    ...overrides
  } as LeaderboardRow
}

function data(scoringRule: LeaderboardData['scoringRule']): LeaderboardData {
  const score = scoringRule === 'score'
  return {
    scoringRule,
    title: score ? '总分榜' : '罚时榜',
    durationSeconds: 7200,
    currentTimeSeconds: 3600,
    solvedLabel: score ? '满分' : '题数',
    metricLabel: score ? '总分' : '罚时',
    metricDirection: score ? 'descending' : 'ascending',
    participantCount: 10,
    awardPercents: { gold: 10, silver: 10, bronze: 10 },
    problems: [{ id: 1, label: 'A', color: '#ff46a0', maxScore: 100 }],
    rows: [score ? row({ metric: 100, metricDisplay: '100/100', maxScore: 100, results: { '1': { status: 'accepted', attempts: 1, primary: '100/100', secondary: '满分' } } }) : row()]
  }
}

function mountBoard(boardData: LeaderboardData) {
  return mount(LeaderboardBoard, {
    props: { data: boardData },
    global: { directives: { loading: () => undefined } }
  })
}

describe('LeaderboardBoard variants', () => {
  it('renders the penalty table with name-only participant identity', () => {
    const wrapper = mountBoard(data('penalty'))

    expect(wrapper.find('[data-scoreboard-branch="penalty"]').exists()).toBe(true)
    expect(wrapper.find('[data-scoreboard-branch="score"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('排名Rank')
    expect(wrapper.text()).toContain('题数Solved')
    expect(wrapper.text()).toContain('罚时Penalty')
    expect(wrapper.text()).toContain('章佳荣')
    expect(wrapper.text()).not.toContain('STUDENT-NUMBER-MUST-NOT-RENDER')
    expect(wrapper.text()).not.toContain('SUBMISSION-METADATA-MUST-NOT-RENDER')
    expect(wrapper.find('.student-avatar').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('学生 / 学号')
    expect(wrapper.text()).not.toContain('团队')
    expect(wrapper.text()).not.toContain('组织')

    expect(wrapper.findAll('.penalty-header-card.native-header-card')).toHaveLength(3)
    expect(wrapper.find('.penalty-problem-header.native-header-card').exists()).toBe(true)
    expect(wrapper.find('.penalty-row.native-stripe-row').exists()).toBe(true)
    expect(wrapper.find('.penalty-rank-stat.is-gold').exists()).toBe(true)
    expect(wrapper.find('.penalty-solved-stat').exists()).toBe(true)
    expect(wrapper.find('.penalty-time-stat').exists()).toBe(true)
    expect(wrapper.find('.penalty-result.result-accepted.is-first').exists()).toBe(true)
    expect(wrapper.text()).toContain('金 10%')
  })

  it('renders the score table and full-score cells', () => {
    const wrapper = mountBoard(data('score'))

    expect(wrapper.find('[data-scoreboard-branch="score"]').exists()).toBe(true)
    expect(wrapper.find('[data-scoreboard-branch="penalty"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('排名Rank')
    expect(wrapper.text()).toContain('满分Full')
    expect(wrapper.text()).toContain('总分Score')
    expect(wrapper.text()).toContain('100/100')
    expect(wrapper.text()).toContain('满分')
  })

  it.each([
    ['penalty', '.penalty-header-track'],
    ['score', '.score-header-track']
  ] as const)('keeps the %s sticky header outside its horizontal body scroller and synchronizes scroll', async (rule, headerSelector) => {
    const wrapper = mountBoard(data(rule))
    const branch = wrapper.get(`[data-scoreboard-branch="${rule}"]`)
    const stickyHeader = wrapper.get(`[data-sticky-header="${rule}"]`)
    const bodyScroller = wrapper.get(`[data-horizontal-scroll="${rule}"]`)

    expect(stickyHeader.element.parentElement).toBe(branch.element)
    expect(bodyScroller.element.parentElement).toBe(branch.element)
    expect(stickyHeader.element.contains(bodyScroller.element)).toBe(false)
    expect(wrapper.find('.rank-scroll-viewport').exists()).toBe(false)

    Object.defineProperty(bodyScroller.element, 'scrollLeft', { configurable: true, value: 128 })
    await bodyScroller.trigger('scroll')

    expect(wrapper.get(headerSelector).attributes('style')).toContain('translate3d(-128px')
  })

  it('uses scoring-rule-specific performance comparators without mutating input', () => {
    const published = [
      row({ id: 1, rank: 1, name: '甲', solved: 2, metric: 90 }),
      row({ id: 2, rank: 2, name: '乙', solved: 1, metric: 100 })
    ]

    expect(sortLeaderboardRows(published, 'score', 'performance').map((item) => item.name)).toEqual(['乙', '甲'])
    expect(sortLeaderboardRows(published, 'penalty', 'performance').map((item) => item.name)).toEqual(['甲', '乙'])
    expect(published.map((item) => item.name)).toEqual(['甲', '乙'])
  })
})
