import { mount, flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import SourceCodeViewer from '../src/components/SourceCodeViewer.vue'
import TeamJoin from '../src/views/teams/TeamJoin.vue'
import { generatedApi } from '../src/api/client'
import { http } from '../src/api/http'
import { getStatementAsset } from '../src/api/modules/assets'

const route = reactive({ params: {} as Record<string, string> })
const router = { push: vi.fn(), replace: vi.fn() }
vi.mock('vue-router', () => ({ useRoute: () => route, useRouter: () => router }))
vi.mock('../src/features/clipboard', () => ({ copyTextToClipboard: vi.fn().mockResolvedValue(undefined) }))
import { copyTextToClipboard } from '../src/features/clipboard'

const stubs = {
  ElButton: { props: ['disabled', 'nativeType'], template: '<button :disabled="disabled" :type="nativeType || \'button\'"><slot /></button>' },
  ElInput: { props: ['modelValue'], emits: ['update:modelValue'], template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />' },
  ElForm: { template: '<form><slot /></form>' }, ElFormItem: { template: '<label><slot /></label>' },
  ElRadioGroup: { template: '<div><slot /></div>' }, ElRadioButton: { template: '<span><slot /></span>' },
  ElAlert: { props: ['title'], template: '<aside>{{ title }}</aside>' },
  ElResult: { props: ['title'], template: '<aside>{{ title }}<slot name="extra" /></aside>' },
}
const team = { id: 10, name: '邀请团队', slug: 'qa-invite', visibility: 'private' as const, join_mode: 'invitation' as const, joined: false, application_status: '' }

describe('治理与共享提交详情', () => {
  beforeEach(() => { vi.restoreAllMocks(); vi.clearAllMocks(); route.params = {} })
  it('复制保留原始空行、制表符和换行，不包括行号；空代码不可复制', async () => {
    const source = '\tprint(3)\r\n\r\nprint(4)\n'
    const wrapper = mount(SourceCodeViewer, { props: { source }, global: { stubs } })
    expect(wrapper.get('.source-lines').text()).toBe('1\n2\n3\n4')
    await wrapper.get('button').trigger('click')
    expect(copyTextToClipboard).toHaveBeenCalledWith(source)
    await wrapper.setProps({ source: '' })
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
  it('只有邀请码即可找到并确认加入私有团队，不要求链接或二次输入', async () => {
    const preview = vi.spyOn(generatedApi, 'previewTeamInvitation').mockResolvedValue({ data: team } as any)
    const join = vi.spyOn(generatedApi, 'joinTeam').mockResolvedValue({ data: { joined: true } } as any)
    const wrapper = mount(TeamJoin, { global: { stubs, directives: { loading: {} } } })
    expect(wrapper.findAll('input')).toHaveLength(1)
    await wrapper.get('input').setValue(' abcdef123456 ')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(preview).toHaveBeenCalledWith({ body: { join_code: 'abcdef123456' } })
    expect(wrapper.text()).toContain('邀请团队')
    expect(wrapper.findAll('input')).toHaveLength(0)
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(join).toHaveBeenCalledWith({ path: { id: 10 }, body: { join_code: 'abcdef123456', message: '' } })
    expect(router.replace).toHaveBeenCalledWith('/teams/qa-invite/contests')
    wrapper.unmount()
  })
  it('申请链接直接进入待审流程，不调用邀请码查询', async () => {
    route.params = { teamSlug: 'application-team' }
    const preview = vi.spyOn(generatedApi, 'previewTeamJoin').mockResolvedValue({ data: { ...team, join_mode: 'application', application_status: 'pending' } } as any)
    const invitation = vi.spyOn(generatedApi, 'previewTeamInvitation')
    const wrapper = mount(TeamJoin, { global: { stubs, directives: { loading: {} } } })
    await flushPromises()
    expect(preview).toHaveBeenCalledWith({ path: { id: 'application-team' } })
    expect(wrapper.text()).toContain('申请审核中')
    expect(invitation).not.toHaveBeenCalled()
    expect(wrapper.find('form').exists()).toBe(false)
    wrapper.unmount()
  })
  it('旧待审记录不阻挡已改为自由加入的团队', async () => {
    route.params = { teamSlug: 'open-team' }
    vi.spyOn(generatedApi, 'previewTeamJoin').mockResolvedValue({ data: { ...team, join_mode: 'open', application_status: 'pending' } } as any)
    const wrapper = mount(TeamJoin, { global: { stubs, directives: { loading: {} } } })
    await flushPromises()
    expect(wrapper.text()).toContain('确认加入')
    expect(wrapper.text()).not.toContain('申请审核中')
    wrapper.unmount()
  })
  it('生成请求保留多段资源路径、二进制与 multipart 请求体', async () => {
    const request = vi.spyOn(http, 'request').mockResolvedValue({ data: new Blob(['ok']) })
    await getStatementAsset(3, 'images/中文 + 图.png')
    expect(request).toHaveBeenLastCalledWith(expect.objectContaining({ url: '/problems/3/assets/images/%E4%B8%AD%E6%96%87%20%2B%20%E5%9B%BE.png', responseType: 'blob' }))
    await getStatementAsset(3, 'images/a.png', '/exams/7/problems/3/assets')
    expect(request).toHaveBeenLastCalledWith(expect.objectContaining({ url: '/exams/7/problems/3/assets/images/a.png' }))
    await expect(getStatementAsset(3, '../secret')).rejects.toThrow()
    expect(() => getStatementAsset(3, 'a.png', '/exams/7/problems/4/assets')).toThrow()
    const body = new FormData(); body.append('file', new Blob(['rows']), 'students.csv')
    await generatedApi.importClassStudents({ path: { id: 3 }, body })
    expect(request).toHaveBeenLastCalledWith(expect.objectContaining({ data: body, method: 'POST', url: '/classes/3/students/import' }))
  })
  it('所有提交详情入口复用同一展示组件', () => {
    for (const path of ['AssignmentDetail.vue', 'Submissions.vue', 'ExamDetail.vue', 'problems/ProblemSubmissions.vue', 'teams/ContestWorkspace.vue', 'teams/TeamProblemSetDetail.vue']) {
      expect(readFileSync(resolve('src/views', path), 'utf8')).toContain('SubmissionDetailDialog')
    }
    expect(readFileSync(resolve('src/views/Exams.vue'), 'utf8')).toContain('SubmissionDetailContent')
  })
})
