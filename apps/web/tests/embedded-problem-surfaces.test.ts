import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ProblemOverview from '../src/components/ProblemOverview.vue'
import ProblemStatementView from '../src/components/ProblemStatementView.vue'

const problem = {
  id: 1,
  title: 'Embedded problem',
  statement: 'Problem statement',
  time_limit_ms: 1000,
  memory_limit_mb: 256,
  output_limit_kb: 1024,
  tags: ''
}

describe('embedded exam and contest problem surfaces', () => {
  it('flattens the shared overview into its parent content area', () => {
    const wrapper = mount(ProblemOverview, {
      props: {
        embedded: true,
        items: [{ problem, label: 'A', score: 100 }]
      },
      global: { stubs: { ElTag: true, ElEmpty: true } }
    })

    expect(wrapper.get('section').classes()).toContain('problem-overview--embedded')
    expect(wrapper.get('.overview-table-wrap').exists()).toBe(true)
  })

  it('uses a full-width shared statement surface without dropping content', () => {
    const wrapper = mount(ProblemStatementView, {
      props: {
        embedded: true,
        problem,
        showMeta: false
      },
      global: { stubs: { MarkdownRenderer: true, ElTag: true } }
    })

    expect(wrapper.get('.problem-view-grid').classes()).toEqual(
      expect.arrayContaining(['problem-view-grid--embedded', 'problem-view-grid--single'])
    )
    expect(wrapper.get('.statement-box').exists()).toBe(true)
    expect(wrapper.find('aside').exists()).toBe(false)
    expect(wrapper.text()).toContain(problem.title)
  })

  it('enables the shared embedded mode at every exam and team contest entry', () => {
    const examOverview = readFileSync(
      resolve(process.cwd(), 'src/views/exam/ExamOverview.vue'),
      'utf8'
    )
    const examProblems = readFileSync(
      resolve(process.cwd(), 'src/views/exam/ExamProblems.vue'),
      'utf8'
    )
    const teamContest = readFileSync(
      resolve(process.cwd(), 'src/views/teams/ContestWorkspace.vue'),
      'utf8'
    )

    expect(examOverview).toMatch(/<ProblemOverview\s+embedded/)
    expect(examProblems).toMatch(/<ProblemStatementView\s+\n?\s*embedded/)
    expect(teamContest).toMatch(/<ProblemStatementView[^>]*\sembedded\s/)
  })
})
