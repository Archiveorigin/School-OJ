import type { Role } from '../../api/client'

export interface NavItem {
  path: string
  label: string
  roles?: Role[]
  hiddenFor?: Role[]
}

export interface NavGroup {
  label: string
  items: NavItem[]
}

const teacherRoles: Role[] = ['admin', 'teacher']

export const navGroups: NavGroup[] = [
  {
    label: '学习',
    items: [
      { path: '/', label: '概览' },
      { path: '/problems', label: '题库' },
      { path: '/assignments', label: '作业' },
      { path: '/exams', label: '考试' },
      { path: '/submissions', label: '提交' },
      { path: '/leaderboard', label: '排行榜' }
    ]
  },
  {
    label: '教学',
    items: [
      { path: '/courses', label: '课程班级' },
      { path: '/prepared-problems', label: '预备题库', roles: teacherRoles },
      { path: '/plagiarism', label: 'JPlag 查重', roles: teacherRoles }
    ]
  },
  {
    label: '系统',
    items: [
      { path: '/admin/exam-rankings', label: '考试实时榜', roles: ['admin'] },
      { path: '/audit-logs', label: '审计日志', roles: ['admin'] },
      { path: '/users', label: '用户管理', roles: ['admin'] }
    ]
  }
]

export function visibleNavGroups(role?: Role) {
  const currentRole = role || 'student'
  return navGroups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => {
        if (item.roles && !item.roles.includes(currentRole)) return false
        if (item.hiddenFor?.includes(currentRole)) return false
        return true
      })
    }))
    .filter((group) => group.items.length > 0)
}
