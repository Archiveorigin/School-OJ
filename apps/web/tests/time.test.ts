import { describe, expect, it } from 'vitest'
import { formatDateTime } from '../src/features/time'

describe('formatDateTime', () => {
  it('formats visible timestamps consistently', () => {
    expect(formatDateTime(new Date(2026, 5, 7, 15, 4, 5))).toBe('2026-06-07 15:04:05')
  })

  it('uses a dash for empty timestamps', () => {
    expect(formatDateTime(null)).toBe('-')
  })
})
