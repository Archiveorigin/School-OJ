import { flushPromises, shallowMount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { client } from '../src/api/client'
import { useAuthStore } from '../src/stores/auth'
import AdminHome from '../src/views/admin/AdminHome.vue'
import AdminLayout from '../src/views/admin/AdminLayout.vue'

const routerMocks = vi.hoisted(() => ({ push: vi.fn() }))

vi.mock('vue-router', () => ({
  useRoute: () => ({ meta: { title: '教学概览', adminMenu: '/admin' } }),
  useRouter: () => ({ push: routerMocks.push })
}))

const globalOptions = {
  stubs: {
    RouterLink: { props: ['to'], template: '<a><slot /></a>' },
    RouterView: true,
    'el-icon': { template: '<span><slot /></span>' },
    'el-button': { template: '<button><slot /></button>' },
    'el-dropdown': { template: '<div><slot /><slot name="dropdown" /></div>' },
    'el-dropdown-menu': { template: '<div><slot /></div>' },
    'el-dropdown-item': { template: '<button><slot /></button>' }
  },
  directives: {
    loading: {}
  }
}

function setUser(role: 'admin' | 'teacher') {
  const auth = useAuthStore()
  auth.$patch({
    token: 'test-token',
    hydrated: true,
    user: {
      id: 7,
      email: `${role}@example.com`,
      name: role === 'admin' ? '系统管理员' : '张老师',
      role,
      can_author: true
    } as any
  })
  return auth
}

beforeEach(() => {
  setActivePinia(createPinia())
  routerMocks.push.mockReset()
  localStorage.clear()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('admin workspace', () => {
  it('shows governance modules only to administrators', () => {
    setUser('admin')
    const adminView = shallowMount(AdminLayout, { global: globalOptions })
    expect(adminView.text()).toContain('用户与权限')
    expect(adminView.text()).toContain('审计日志')
    expect(adminView.text()).toContain('工单管理')

    setUser('teacher')
    const teacherView = shallowMount(AdminLayout, { global: globalOptions })
    expect(teacherView.text()).not.toContain('用户与权限')
    expect(teacherView.text()).not.toContain('审计日志')
    expect(teacherView.text()).not.toContain('工单管理')
    expect(teacherView.text()).toContain('课程管理')
    expect(teacherView.text()).toContain('考试管理')
    expect(teacherView.text()).toContain('JPlag 查重')
  })

  it('maps teaching metrics, upcoming exams and administrator queues into the overview', async () => {
    setUser('admin')
    vi.spyOn(client, 'get').mockImplementation(async (url) => {
      if (url === '/exams')
        return {
          data: [
            {
              id: 8,
              title: '算法设计期中考试',
              course_code: 'CS2026',
              course_name: '算法设计',
              class_name: '计算机 1 班',
              starts_at: '2099-09-05T01:00:00.000Z',
              ends_at: '2099-09-05T05:00:00.000Z'
            }
          ]
        } as any
      if (url === '/courses') return { data: [{ id: 2, code: 'CS2026', name: '算法设计' }] } as any
      if (url === '/classes')
        return {
          data: [{ id: 4, class_id: 4, course_id: 2, class_name: '计算机 1 班' }]
        } as any
      if (url === '/author-applications') return { data: [{ id: 21, status: 'pending' }] } as any
      if (url === '/problem-change-tickets') return { data: [{ id: 22, status: 'pending' }] } as any
      if (url === '/audit-logs')
        return {
          data: [
            {
              id: 31,
              actor_name: '王志华',
              action: 'exam.create',
              resource_label: '考试',
              created_at: '2026-08-30T01:10:00.000Z'
            }
          ]
        } as any
      throw new Error(`unexpected endpoint: ${url}`)
    })

    const wrapper = shallowMount(AdminHome, { global: globalOptions })
    await flushPromises()

    expect(wrapper.text()).toContain('教学概览')
    expect(wrapper.text()).toContain('算法设计期中考试')
    expect(wrapper.text()).toContain('CS2026')
    expect(wrapper.text()).toContain('计算机 1 班')
    expect(wrapper.text()).toContain('出题权限申请')
    expect(wrapper.text()).toContain('题目修改工单')
    expect(wrapper.text()).toContain('王志华')
    expect(wrapper.text()).toContain('创建了考试')
    expect(client.get).toHaveBeenCalledWith('/author-applications', {
      params: { status: 'pending' }
    })
    expect(client.get).toHaveBeenCalledWith('/problem-change-tickets', {
      params: { status: 'pending' }
    })

    const primaryAction = wrapper.findAll('.heading-actions button').at(-1)
    expect(primaryAction?.text()).toContain('新建考试')
    await primaryAction?.trigger('click')
    expect(routerMocks.push).toHaveBeenCalledWith('/admin/exams/new')
  })

  it('does not call administrator-only endpoints for teachers', async () => {
    setUser('teacher')
    const get = vi.spyOn(client, 'get').mockImplementation(async (url) => {
      if (url === '/exams' || url === '/courses' || url === '/classes') return { data: [] } as any
      throw new Error(`teacher requested forbidden endpoint: ${url}`)
    })

    shallowMount(AdminHome, { global: globalOptions })
    await flushPromises()

    expect(get).toHaveBeenCalledTimes(3)
    expect(get).not.toHaveBeenCalledWith('/audit-logs')
    expect(get).not.toHaveBeenCalledWith('/author-applications', expect.anything())
    expect(get).not.toHaveBeenCalledWith('/problem-change-tickets', expect.anything())
  })
})
