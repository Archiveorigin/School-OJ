import { afterEach, describe, expect, it, vi } from 'vitest'
import { randomUUID } from '../src/features/uuid'

afterEach(() => vi.unstubAllGlobals())
describe('UUID generation across browser contexts', () => {
  it('uses the native API with its Crypto receiver when available', () => {
    const source = { randomUUID() { expect(this).toBe(source); return 'native-uuid' }, getRandomValues: vi.fn() }
    vi.stubGlobal('crypto', source)
    expect(randomUUID()).toBe('native-uuid')
    expect(source.getRandomValues).not.toHaveBeenCalled()
  })
  it.each([0, 255])('uses secure random bytes and correct v4/variant bits without randomUUID (%s)', value => {
    const source = { getRandomValues(bytes: Uint8Array) { expect(this).toBe(source); return bytes.fill(value) } }
    vi.stubGlobal('crypto', source)
    expect(randomUUID()).toBe(value === 0
      ? '00000000-0000-4000-8000-000000000000'
      : 'ffffffff-ffff-4fff-bfff-ffffffffffff')
  })
  it('reports an unsupported browser instead of generating a weak identifier', () => {
    vi.stubGlobal('crypto', undefined)
    expect(() => randomUUID()).toThrow('当前浏览器不支持安全随机数')
  })
})
