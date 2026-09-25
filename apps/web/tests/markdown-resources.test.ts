import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import MarkdownRenderer from '../src/components/MarkdownRenderer.vue'
import { getStatementAsset } from '../src/api/modules/assets'
import { markdown } from '../src/features/markdown/renderer'

vi.mock('../src/api/modules/assets', () => ({ getStatementAsset: vi.fn() }))
beforeEach(() => {
  vi.clearAllMocks()
  let id = 0
  vi.stubGlobal('URL', class extends URL {
    static createObjectURL = vi.fn(() => `blob:test-${++id}`)
    static revokeObjectURL = vi.fn()
  })
})
afterEach(() => vi.unstubAllGlobals())

describe('Markdown resource lifecycle', () => {
  it('resolves encoded image names without double-encoding attachment paths', async () => {
    vi.mocked(getStatementAsset).mockResolvedValue({ data: new Blob(['image']) } as any)
    const wrapper = mount(MarkdownRenderer, { props: { source: '![图](assets/%E5%9B%BE%201.png)', problemId: 1 } })
    await flushPromises()
    expect(getStatementAsset).toHaveBeenCalledWith(1, 'assets/图 1.png', undefined, expect.any(AbortSignal))
    expect(wrapper.get('img').attributes('src')).toBe('blob:test-1')
    wrapper.unmount()
  })
  it('parses once per source and does not reload unchanged asset paths on text edits', async () => {
    vi.mocked(getStatementAsset).mockResolvedValue({ data: new Blob(['image']) } as any)
    const parse = vi.spyOn(markdown, 'parse')
    const wrapper = mount(MarkdownRenderer, { props: { source: '![x](assets/a.png) $a^2$', problemId: 1 } })
    await flushPromises()
    expect(parse).toHaveBeenCalledTimes(1)
    expect(wrapper.get('img').attributes('src')).toBe('blob:test-1')
    await wrapper.setProps({ source: 'new text ![x](assets/a.png) $a^2$' })
    await flushPromises()
    expect(getStatementAsset).toHaveBeenCalledTimes(1)
    expect(wrapper.get('img').attributes('src')).toBe('blob:test-1')
    expect(wrapper.find('.katex').exists()).toBe(true)
    wrapper.unmount()
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:test-1')
    parse.mockRestore()
  })
  it('revokes partially loaded assets and aborts pending reads on unmount', async () => {
    let resolveSecond!: (value: any) => void
    vi.mocked(getStatementAsset).mockImplementation((_id, path) => path.endsWith('a.png')
      ? Promise.resolve({ data: new Blob(['a']) } as any)
      : new Promise(resolve => { resolveSecond = resolve }))
    const wrapper = mount(MarkdownRenderer, { props: { source: '![a](assets/a.png) ![b](assets/b.png)', problemId: 1 } })
    await flushPromises()
    expect(URL.createObjectURL).toHaveBeenCalledTimes(1)
    const signal = vi.mocked(getStatementAsset).mock.calls[1][3]!
    wrapper.unmount()
    expect(signal.aborted).toBe(true)
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:test-1')
    resolveSecond({ data: new Blob(['b']) })
    await flushPromises()
    expect(URL.createObjectURL).toHaveBeenCalledTimes(1)
  })
  it('honors supplied URLs and isolates resource scopes between instances', async () => {
    vi.mocked(getStatementAsset).mockResolvedValue({ data: new Blob(['image']) } as any)
    const a = mount(MarkdownRenderer, { props: { source: '![a](assets/a.png)', problemId: 1, assetUrls: { 'assets/a.png': 'blob:provided' } } })
    const b = mount(MarkdownRenderer, { props: { source: '![a](assets/a.png)', problemId: 2 } })
    await flushPromises()
    expect(getStatementAsset).toHaveBeenCalledTimes(1)
    expect(a.get('img').attributes('src')).toBe('blob:provided')
    expect(b.get('img').attributes('src')).toBe('blob:test-1')
    await b.setProps({ assetBase: '/exams/4/problems/2/assets' })
    await flushPromises()
    expect(getStatementAsset).toHaveBeenCalledTimes(2)
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:test-1')
    a.unmount(); b.unmount()
    expect(URL.revokeObjectURL).not.toHaveBeenCalledWith('blob:provided')
  })
})
