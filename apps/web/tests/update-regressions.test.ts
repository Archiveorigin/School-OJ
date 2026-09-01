import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { client } from '../src/api/client'
import PermissionManagement from '../src/views/admin/PermissionManagement.vue'

const source = (file: string) => readFileSync(resolve(process.cwd(), file), 'utf8')

afterEach(() => vi.restoreAllMocks())

describe('second-round UI regressions', () => {
  it('uses the popup algorithm selector in the catalog', () => {
    const catalog = source('src/views/problems/ProblemCatalog.vue')
    expect(catalog).toContain('<ProblemTagSelector v-model="filters.tags"')
    expect(catalog).not.toMatch(/<el-select[^>]+v-model="filters\.tags"/)
  })

  it('reuses the exam leaderboard surface in team contests and removes the bespoke matrix', () => {
    const workspace = source('src/views/teams/ContestWorkspace.vue')
    expect(workspace).toContain('<LeaderboardBoard')
    expect(workspace).toContain('adaptTeamContestRanking')
    expect(workspace).not.toContain('ContestScoreCell')
    expect(workspace).not.toContain('scoreboard-scroll')
  })

  it('keeps base roles visible while displaying the independent author permission', async () => {
    vi.spyOn(client, 'get').mockImplementation(async (url) => {
      if (url === '/users')
        return {
          data: [
            {
              id: 2,
              name: '林同学',
              email: 'lin@example.com',
              role: 'student',
              can_author: true
            }
          ]
        } as any
      if (url === '/author-applications') return { data: [] } as any
      if (url === '/permissions')
        return {
          data: [
            {
              key: 'problem_author',
              name: '出题者',
              description: '独立出题权限',
              scope: 'global'
            }
          ]
        } as any
      throw new Error(`unexpected endpoint: ${url}`)
    })
    const wrapper = shallowMount(PermissionManagement, {
      global: {
        directives: { loading: {} },
        stubs: {
          'el-button': { template: '<button><slot /></button>' },
          'el-tag': { template: '<span><slot /></span>' },
          'el-table': { template: '<div><slot /></div>' },
          'el-table-column': true,
          'el-empty': true,
          'el-input': true,
          'el-select': { template: '<select><slot /></select>' },
          'el-option': true
        }
      }
    })
    await flushPromises()
    expect(wrapper.text()).toContain('附加权限不会覆盖')
    expect(wrapper.text()).toContain('出题者')
    expect(wrapper.text()).toContain('独立出题权限')
  })

  it('registers permission management as a user-and-permissions child route', () => {
    const router = source('src/router/index.ts')
    expect(router).toMatch(/path:\s*["']users["']/)
    expect(router).toMatch(/path:\s*["']permissions["']/)
    expect(router).toMatch(/redirect:\s*["']\/admin\/users\/permissions["']/)
  })
})
