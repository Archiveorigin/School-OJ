import type { Role } from '../../api/client'

export interface NavItem {
  path: string
  label: string
  roles?: Role[]
  requiresAuth?: boolean
  authorOnly?: boolean
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
      { path: '/problems/create', label: '出题', authorOnly: true, requiresAuth: true },
      { path: '/my/courses', label: '课程', requiresAuth: true }
    ]
  }
]

export function visibleNavGroups(role?: Role, authenticated = false, canAuthor = false) {
  return navGroups.map((group) => ({
    ...group,
    items: group.items.filter((item) => {
      if (item.requiresAuth && !authenticated) return false
      if (item.authorOnly && !canAuthor) return false
      return !item.roles || (role ? item.roles.includes(role) : false)
    })
  }))
}
