import { describe, expect, it } from 'vitest'
import router from '../src/router'

describe('admin exam navigation', () => {
  it('keeps exam management and authoring inside the admin workspace', () => {
    const list = router.resolve('/admin/exams')
    const create = router.resolve('/admin/exams/new?course_id=9')

    expect(list.matched.map((item) => item.path)).toEqual(['/admin', '/admin/exams'])
    expect(list.meta.adminMenu).toBe('/admin/exams')
    expect(create.matched.map((item) => item.path)).toEqual(['/admin', '/admin/exams/new'])
    expect(create.meta.roles).toEqual(['admin', 'teacher'])
    expect(create.meta.adminMenu).toBe('/admin/exams')
    expect(create.query.course_id).toBe('9')
  })

  it('retains the legacy exam authoring entry as a redirect', () => {
    const legacy = router.getRoutes().find((item) => item.path === '/exams/new')
    expect(legacy?.redirect).toBeTypeOf('function')
  })
})
