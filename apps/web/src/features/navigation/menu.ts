import type { Role } from '../../api/client'

export interface NavItem {
  path: string
  label: string
  roles?: Role[]
  requiresAuth?: boolean
}

export interface NavGroup {
  label: string
  items: NavItem[]
}

export const navGroups: NavGroup[] = [
  {
    label: '公共功能',
    items: [
      { path: '/', label: '概览' },
      { path: '/problems', label: '题库' },
      { path: '/my/courses', label: '课程', requiresAuth: true }
    ]
  }
]

export function visibleNavGroups(role?: Role, authenticated = false) {
  return navGroups.map((group) => ({
    ...group,
    items: group.items.filter((item) => {
      if (item.requiresAuth && !authenticated) return false
      return !item.roles || (role ? item.roles.includes(role) : false)
    })
  }))
}
