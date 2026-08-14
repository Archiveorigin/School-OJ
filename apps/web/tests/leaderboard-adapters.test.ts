import { describe, expect, it } from 'vitest'
import { adaptExamRanking, adaptTeamContestRanking } from '../src/features/leaderboard/adapters'

describe('leaderboard adapters', () => {
  it('maps a penalty contest with name-only identity and ICPC cells', () => {
    const data = adaptTeamContestRanking({
      contest: {
        title: '春季赛', duration_minutes: 120, starts_at: '2026-08-07T08:00:00Z',
        gold_award_percent: 5, silver_award_percent: 10, bronze_award_percent: 15
      },
      scoring_rule: 'penalty',
      problems: [{ problem_id: 10, label: 'A', title: '相加' }],
      rows: [{
        user_id: 7,
        name: '参赛者甲',
        student_no: '20260007',
        solved: 1,
        penalty_minutes: 42,
        submission_count: 2,
        problems: [{ problem_id: 10, status: 'running', solved_at: '2026-08-07T08:22:00Z', attempts: 2, elapsed_minutes: 22, fastest: true }]
      }]
    }, { team: { name: '不应显示的算法队' } })

    expect(data.scoringRule).toBe('penalty')
    expect(data.metricLabel).toBe('罚时')
    expect(data.solvedLabel).toBe('题数')
    expect(data.metricDirection).toBe('ascending')
    expect(data.awardPercents).toEqual({ gold: 5, silver: 10, bronze: 15 })
    expect(data.participantCount).toBe(1)
    expect(data.rows[0]).toMatchObject({ rank: 1, name: '参赛者甲', solved: 1, metric: 42, metricDisplay: '42' })
    expect(data.rows[0]).not.toHaveProperty('studentNo')
    expect(data.rows[0]).not.toHaveProperty('meta')
    expect(data.rows[0]).not.toHaveProperty('organization')
    expect(data.rows[0].results['10']).toMatchObject({ status: 'accepted', attempts: 2, firstBlood: true, primary: '2', secondary: "22'" })
  })

  it('maps a score contest to fixed 100-point problem cells', () => {
    const data = adaptTeamContestRanking({
      scoring_rule: 'score',
      contest: { title: '积分赛', duration_minutes: 90 },
      problems: [{ problem_id: 1, label: 'A' }, { problem_id: 2, label: 'B' }],
      rows: [{
        user_id: 8,
        name: '参赛者乙',
        student_no: '20260008',
        solved: 1,
        total_score: 160,
        submission_count: 3,
        problems: [
          { problem_id: 1, status: 'running', attempts: 2, best_score: 100 },
          { problem_id: 2, status: 'wrong_answer', attempts: 2, best_score: 60 }
        ]
      }]
    })

    expect(data).toMatchObject({ scoringRule: 'score', solvedLabel: '满分', metricLabel: '总分' })
    expect(data.rows[0]).toMatchObject({ name: '参赛者乙', metric: 160, maxScore: 200, metricDisplay: '160/200' })
    expect(data.rows[0]).not.toHaveProperty('studentNo')
    expect(data.rows[0].results['1']).toMatchObject({ status: 'accepted', primary: '100/100', secondary: '满分' })
    expect(data.rows[0].results['2']).toMatchObject({ status: 'wrong', primary: '60/100', secondary: '未满分' })
  })

  it('keeps full-score cells accepted when newer judging or review work is pending', () => {
    const data = adaptExamRanking({
      scoring_rule: 'score',
      exam: { title: '组合状态考试' },
      problems: [
        { problem_id: 1, label: 'A', score: 100 },
        { problem_id: 2, label: 'B', score: 100 },
        { problem_id: 3, label: 'C', score: 100 }
      ],
      rows: [{
        rank: 1,
        user_id: 18,
        name: '满分优先学生',
        student_no: '20260018',
        solved: 2,
        total_score: 260,
        max_score: 300,
        submission_count: 6,
        pending_count: 2,
        problems: [
          { problem_id: 1, status: 'running', attempts: 2, best_score: 100 },
          { problem_id: 2, status: 'queued', attempts: 2, best_score: 100, pending: true, score_ready: false },
          { problem_id: 3, status: 'accepted', attempts: 2, best_score: 60, pending: true, score_ready: false }
        ]
      }]
    })

    expect(data.rows[0].solved).toBe(2)
    expect(data.rows[0].results['1']).toMatchObject({ status: 'accepted', primary: '100/100', secondary: '满分' })
    expect(data.rows[0].results['2']).toMatchObject({ status: 'accepted', primary: '100/100', secondary: '满分' })
    expect(data.rows[0].results['3']).toMatchObject({ status: 'pending', primary: '60/100', secondary: '待评分' })
  })

  it('maps an exam score board, pending reviews and name-only identity', () => {
    const data = adaptExamRanking({
      scoring_rule: 'score',
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
    expect(data.rows[0]).toMatchObject({ name: '学生乙', metric: 60, metricDisplay: '60/100' })
    expect(data.rows[0]).not.toHaveProperty('studentNo')
    expect(data.rows[0]).not.toHaveProperty('meta')
    expect(data.rows[0]).not.toHaveProperty('organization')
    expect(data.rows[0].results['20']).toMatchObject({ status: 'pending', primary: '60/100', secondary: '待评分', timeSeconds: 1800 })
  })

  it('maps an exam penalty board and safely falls back invalid rules', () => {
    const data = adaptExamRanking({
      scoring_rule: 'penalty',
      exam: { title: '算法考试', starts_at: '2026-08-07T08:00:00Z', ends_at: '2026-08-07T10:00:00Z' },
      problems: [{ problem_id: 30, label: 'A', score: 50 }],
      rows: [{
        rank: 1,
        user_id: 10,
        name: '学生丙',
        student_no: '20260010',
        solved: 1,
        penalty_minutes: 75,
        submission_count: 3,
        problems: [{ problem_id: 30, status: 'accepted', attempts: 3, wrong_attempts: 2, elapsed_minutes: 35, fastest: true }]
      }]
    })

    expect(data).toMatchObject({ scoringRule: 'penalty', solvedLabel: '题数', metricLabel: '罚时' })
    expect(data.rows[0]).toMatchObject({ name: '学生丙', metric: 75, metricDisplay: '75' })
    expect(data.rows[0]).not.toHaveProperty('studentNo')
    expect(data.rows[0].results['30']).toMatchObject({ status: 'accepted', attempts: 3, timeSeconds: 2100, firstBlood: true, primary: '3', secondary: "35'" })

    expect(adaptExamRanking({ scoring_rule: 'viewer-choice', rows: [], problems: [] }).scoringRule).toBe('penalty')
  })
})
