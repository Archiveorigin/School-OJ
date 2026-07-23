import type { Role } from '../../api/client'

export interface NavItem {
  path: string
  label: string
  roles?: Role[]
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
      { path: '/problems', label: '题库' }
    ]
  }
]

export function visibleNavGroups(role?: Role) {
  return navGroups.map((group) => ({
    ...group,
    items: group.items.filter((item) => !item.roles || (role ? item.roles.includes(role) : false))
  }))
}
