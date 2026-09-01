import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import TeamContestAwardFields from '../src/components/TeamContestAwardFields.vue'
import {
  contestAwardTotal,
  contestAwardValidationError,
  defaultContestAwardPercents
} from '../src/features/teams/contestAwards'

const source = (path: string) => readFileSync(resolve(process.cwd(), path), 'utf8')

describe('team content management', () => {
  it('validates integer award allocations and the cumulative percentage', () => {
    const defaults = defaultContestAwardPercents()
    expect(contestAwardTotal(defaults)).toBe(30)
    expect(contestAwardValidationError(defaults)).toBe('')
    expect(contestAwardValidationError({ ...defaults, gold_award_percent: 10.5 })).toContain('整数')
    expect(
      contestAwardValidationError({
        gold_award_percent: 40,
        silver_award_percent: 40,
        bronze_award_percent: 30
      })
    ).toContain('不能超过 100%')
  })

  it('shows the live award total and emits individual percentage changes', async () => {
    const wrapper = mount(TeamContestAwardFields, {
      props: {
        goldAwardPercent: 10,
        silverAwardPercent: 10,
        bronzeAwardPercent: 10
      },
      global: {
        stubs: {
          ElInputNumber: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template:
              '<button class="number-input" @click="$emit(\'update:modelValue\', Number(modelValue) + 1)">{{ modelValue }}</button>'
          }
        }
      }
    })

    expect(wrapper.text()).toContain('30%')
    await wrapper.findAll('.number-input')[0].trigger('click')
    expect(wrapper.emitted('update:goldAwardPercent')).toEqual([[11]])

    await wrapper.setProps({ goldAwardPercent: 90 })
    expect(wrapper.text()).toContain('110%')
    expect(wrapper.get('.award-error').text()).toContain('不能超过 100%')
  })

  it('wires contest list management and the active contest workspace', () => {
    const list = source('src/views/teams/TeamContests.vue')
    const detail = source('src/views/teams/ContestWorkspace.vue')
    expect(list).toContain('@click.stop="editContest(row)"')
    expect(list).toContain('@click.stop="deleteContest(row)"')
    expect(list).toContain("query: { manage: 'edit' }")
    expect(list).toContain('await client.delete(`/contests/${row.id}`)')
    expect(detail).toContain('await client.put(`/contests/${contestID.value}`, editForm)')
    expect(detail).toContain('v-if="detail.can_edit"')
    expect(detail).toContain('@click="addVisible = true"')
    expect(detail).toContain('<LeaderboardBoard')
    expect(detail).not.toContain('ContestScoreCell')
  })

  it('keeps manager actions separate from organizer permission', () => {
    const contests = source('src/views/teams/TeamContests.vue')
    const problemSets = source('src/views/teams/TeamProblemSets.vue')

    expect(contests).toContain('v-if="row.can_edit"')
    expect(contests).toContain('v-if="row.can_delete"')
    expect(contests).not.toContain('canOrganize && row.can_edit')
    expect(contests).not.toContain('canOrganize && row.can_delete')
    expect(contests).toContain('!canOrganize.value && !row.can_edit && !row.can_delete')

    expect(problemSets).toContain('v-if="item.can_edit"')
    expect(problemSets).toContain('v-if="item.can_delete"')
    expect(problemSets).not.toContain('canOrganize && item.can_edit')
    expect(problemSets).not.toContain('canOrganize && item.can_delete')
  })

  it('wires problem-set edit and confirmed deletion in lists and details', () => {
    const list = source('src/views/teams/TeamProblemSets.vue')
    const detail = source('src/views/teams/TeamProblemSetDetail.vue')
    expect(list).toContain('@click.stop="openEdit(item)"')
    expect(list).toContain('@click.stop="deleteSet(item)"')
    expect(list).toContain('await client.put(`/problem-sets/${editingItem.value.id}`, editForm)')
    expect(list).toContain('await client.delete(`/problem-sets/${item.id}`)')
    expect(detail).toContain('编辑题单信息')
    expect(detail).toContain('await client.put(`/problem-sets/${problemSetID.value}`, editForm)')
    expect(detail).toContain('await client.delete(`/problem-sets/${problemSetID.value}`)')
    expect(detail).toContain('await router.replace(detail.value?.team?.slug')
  })
})
