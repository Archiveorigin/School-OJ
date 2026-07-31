import { describe, expect, it } from 'vitest'
import { visibleNavGroups } from '../src/features/navigation/menu'

function visiblePaths(canAuthor: boolean) {
  return visibleNavGroups('student', true, canAuthor)
    .flatMap((group) => group.items)
    .map((item) => item.path)
}

describe('author navigation', () => {
  it('keeps authoring hidden for ordinary students', () => {
    expect(visiblePaths(false)).not.toContain('/problems/create')
  })

  it('shows authoring when the independent permission is granted', () => {
    expect(visiblePaths(true)).toContain('/problems/create')
  })

  it('shows teams to every authenticated role', () => {
    expect(visiblePaths(false)).toContain('/teams')
  })
})
