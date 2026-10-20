import { afterEach, describe, expect, it, vi } from 'vitest'
import { importEventMarkdown } from '../src/features/events/documentImport'
import { EVENT_CHUNK_SIZE, uploadEventTests } from '../src/features/events/testUploads'
import { buildChunkPlan } from '../src/features/problems/testFileUpload'
import { generatedApi } from '../src/api/client'

vi.mock('../src/api/client', () => ({ generatedApi: { uploadEventTestChunk: vi.fn().mockResolvedValue({}) } }))
const getRandomValues = globalThis.crypto.getRandomValues.bind(globalThis.crypto)
const v4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/
afterEach(() => { vi.unstubAllGlobals(); vi.clearAllMocks() })

describe('authoring without secure-context-only APIs', () => {
  it('imports a single Markdown problem while preserving its sample and remote image', () => {
    vi.stubGlobal('crypto', { getRandomValues })
    const source = '# 单题\n\n## 题目描述\n![图](https://example.com/image.png)\n\n## 输入格式\n整数\n\n## 输出格式\n整数\n\n## 样例 1 · 输入\n```\n1\n```\n\n## 样例 1 · 输出\n```\n2\n```'
    const first = importEventMarkdown(source), second = importEventMarkdown(source)
    expect(first.key).toMatch(v4)
    expect(second.key).not.toBe(first.key)
    expect(first.sections.samples).toEqual([{ input: '1', output: '2' }])
    expect(first.sections.description).toContain('https://example.com/image.png')
  })
  it('uploads a new problem test pair with stable IDs across chunks on HTTP', async () => {
    vi.stubGlobal('crypto', { getRandomValues })
    const files = [new File([new Uint8Array(EVENT_CHUNK_SIZE + 17)], 'large.in'), new File(['2\n'], 'large.out')]
    const progress = vi.fn()
    const refs = await uploadEventTests(files, 'contest', 7, 3, progress, new AbortController().signal)
    expect(refs.map(ref => ref.chunk_count)).toEqual([2, 1])
    expect(new Set(refs.map(ref => ref.id)).size).toBe(2)
    refs.forEach(ref => expect(ref.id).toMatch(v4))
    const calls = vi.mocked(generatedApi.uploadEventTestChunk).mock.calls.map(([options]) => options!)
    expect(calls.map(call => call.path)).toEqual([
      { kind: 'contest', event_id: 7, upload_id: refs[0].id, chunk_index: 0 },
      { kind: 'contest', event_id: 7, upload_id: refs[0].id, chunk_index: 1 },
      { kind: 'contest', event_id: 7, upload_id: refs[1].id, chunk_index: 0 },
    ])
    expect(calls.map(call => (call.body as Blob).size)).toEqual([EVENT_CHUNK_SIZE, 17, 2])
    expect(progress).toHaveBeenLastCalledWith(100)
    const plan = buildChunkPlan(files)
    plan.references.forEach(ref => expect(ref.id).toMatch(v4))
  })
})
