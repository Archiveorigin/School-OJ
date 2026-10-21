import { describe, expect, it } from 'vitest'
import { hasCapability } from '../src/features/authorization'
import type { User } from '../src/api/client'

describe('server capability snapshots', () => {
  it('fails closed for missing grants and never infers from admin or author labels', () => {
    const user = { id: 1, name: 'admin', email: 'a@example.test', role: 'admin', can_author: true } as User
    expect(hasCapability(user, 'user:manage')).toBe(false)
    expect(hasCapability(user, 'problem:author')).toBe(false)
    expect(hasCapability(null, 'user:manage')).toBe(false)
  })
  it('keeps author rights independent from identity and observes revoked snapshots', () => {
    const user = { id: 1, name: 'student', email: 'a@example.test', role: 'student', permissions: ['problem:author'] } as User
    expect(hasCapability(user, 'problem:author')).toBe(true)
    expect(hasCapability(user, 'user:manage')).toBe(false)
    expect(hasCapability({ ...user, permissions: [] }, 'problem:author')).toBe(false)
    expect(hasCapability({ ...user, account_deleted: true }, 'problem:author')).toBe(false)
  })
})
