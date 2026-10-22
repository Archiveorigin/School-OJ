import { describe, expect, it } from 'vitest'
import { useEditSnapshot } from '../src/features/events/editSnapshot'
import { awardCountToPercent, awardCountsFromPercents } from '../src/features/teams/contestAwards'
import { awardCutoffs } from '../src/features/leaderboard/awards'
import { mount } from '@vue/test-utils'
import TeamContestAwardFields from '../src/components/TeamContestAwardFields.vue'

describe('contest settings edits', () => {
  it('preserves unsaved text across refreshes and unrelated revisions, and retains the stale token for conflicting changes', () => {
    const draft = useEditSnapshot({ title: '', description: '' })
    draft.begin({ title: '比赛', description: '原文' }, 3)
    draft.form.description = '尚未保存的说明\n第二行'
    draft.sync({ title: '比赛', description: '原文' }, 4)
    expect(draft.form.description).toBe('尚未保存的说明\n第二行')
    expect(draft.revision.value).toBe(4)
    draft.sync({ title: '比赛', description: '另一位组织者已保存' }, 5)
    expect(draft.form.description).toBe('尚未保存的说明\n第二行')
    expect(draft.revision.value).toBe(4)
    expect(draft.dirty.value).toBe(true)
  })

  it('accepts refreshed data after a successful save without generating a local revision', () => {
    const draft = useEditSnapshot({ description: '' })
    draft.begin({ description: '原文' }, 1)
    draft.form.description = '更新'
    draft.markSaved()
    expect(draft.revision.value).toBe(1)
    draft.sync({ description: '更新' }, 2)
    expect(draft.dirty.value).toBe(false)
    expect(draft.revision.value).toBe(2)
  })
})

describe('award number conversion', () => {
  it('round trips individual allocations without accidentally awarding an extra seat', () => {
    for (const base of [1, 3, 6, 7, 99, 100, 999, 1000000]) {
      for (const counts of [[1, 0, 0], [0, 1, 0], [0, 0, base], [Math.floor(base / 3), Math.floor(base / 3), base - 2 * Math.floor(base / 3)]]) {
        const [gold, silver, bronze] = counts.map(n => awardCountToPercent(n, base))
        const values = { gold_award_percent: gold, silver_award_percent: silver, bronze_award_percent: bronze }
        expect(awardCountsFromPercents(values, base), `${base}: ${counts}`).toEqual(counts)
        expect(awardCutoffs(base, { gold, silver, bronze })).toMatchObject({ gold: counts[0], silver: counts[0] + counts[1], bronze: counts.reduce((a, b) => a + b) })
      }
    }
  })

  it('switches display units repeatedly without rewriting stored percentages', async () => {
    const wrapper = mount(TeamContestAwardFields, {
      props: { goldAwardPercent: 16.666666, silverAwardPercent: 0, bronzeAwardPercent: 16.666666 },
      global: { stubs: { ElRadioGroup: { name: 'UnitSwitch', props: ['modelValue'], emits: ['update:modelValue'], template: '<div><slot /></div>' }, ElInputNumber: true, ElRadioButton: true } },
    })
    for (const unit of ['count', 'percent', 'count', 'percent']) {
      wrapper.getComponent({ name: 'UnitSwitch' }).vm.$emit('update:modelValue', unit)
      await wrapper.vm.$nextTick()
    }
    expect(wrapper.emitted('update:goldAwardPercent')).toBeUndefined()
    expect(wrapper.emitted('update:silverAwardPercent')).toBeUndefined()
    expect(wrapper.emitted('update:bronzeAwardPercent')).toBeUndefined()
    expect(wrapper.emitted('validity-change')?.at(-1)).toEqual([true])
    wrapper.unmount()
  })
})
