import { describe, expect, it } from 'vitest'
import { adaptExamRanking, adaptTeamContestRanking } from '../src/features/leaderboard/adapters'

describe('leaderboard adapters', () => {
  it('maps a penalty contest without changing its published order', () => {
    const data = adaptTeamContestRanking({
      contest: { title: '春季赛', duration_minutes: 120, starts_at: '2026-08-07T08:00:00Z' },
      scoring_rule: 'penalty',
      problems: [{ problem_id: 10, label: 'A', title: '相加' }],
      rows: [{
        user_id: 7,
        name: '参赛者甲',
        solved: 1,
        penalty_minutes: 42,
        submission_count: 2,
        problems: [{ problem_id: 10, status: 'accepted', attempts: 2, elapsed_minutes: 22, fastest: true }]
      }]
    }, { team: { name: '算法队' } })

    expect(data.metricLabel).toBe('罚时')
    expect(data.metricDirection).toBe('ascending')
    expect(data.rows[0]).toMatchObject({ rank: 1, organization: '算法队', solved: 1, metric: 42 })
    expect(data.rows[0].results['10']).toMatchObject({ status: 'accepted', attempts: 2, firstBlood: true, primary: '2', secondary: "22'" })
  })

  it('maps exam scores, pending reviews and student identity', () => {
    const data = adaptExamRanking({
      exam: {
        title: '期末考试',
        course_name: '程序设计',
        class_name: '计科一班',
        starts_at: '2026-08-07T08:00:00Z',
        ends_at: '2026-08-07T10:00:00Z'
      },
      now: '2026-08-07T09:00:00Z',
      problems: [{ problem_id: 20, label: 'A', score: 100 }],
      rows: [{
        rank: 1,
        user_id: 9,
        name: '学生乙',
        student_no: '20260001',
        total_score: 60,
        max_score: 100,
        solved: 0,
        submission_count: 1,
        pending_count: 1,
        problems: [{ problem_id: 20, best_score: 60, max_score: 100, pending: true, score_ready: false, status: 'accepted', submitted_at: '2026-08-07T08:30:00Z' }]
      }]
    })

    expect(data.durationSeconds).toBe(7200)
    expect(data.currentTimeSeconds).toBe(3600)
    expect(data.rows[0]).toMatchObject({ organization: '学号 20260001', metric: 60, metricDisplay: '60/100' })
    expect(data.rows[0].results['20']).toMatchObject({ status: 'pending', primary: '…', secondary: '待评分', timeSeconds: 1800 })
  })
})
