import { describe, expect, it } from 'vitest'
import {
  TEST_UPLOAD_CHUNK_SIZE,
  buildChunkPlan,
  formatFileSize,
  shouldUseChunkedUpload,
  totalTestFileSize
} from '../src/features/problems/testFileUpload'

describe('large test file upload planning', () => {
  it('splits large files into bounded chunks without reading the file body', () => {
    const file = new File([new Uint8Array(TEST_UPLOAD_CHUNK_SIZE * 2 + 17)], 'tests.zip')
    const plan = buildChunkPlan([file])
    expect(plan.references[0]).toMatchObject({ name: 'tests.zip', chunk_count: 3, size: file.size })
    expect(plan.tasks.map((task) => task.end - task.start)).toEqual([TEST_UPLOAD_CHUNK_SIZE, TEST_UPLOAD_CHUNK_SIZE, 17])
  })

  it('uses chunk mode only for large aggregate uploads', () => {
    const small = new File([new Uint8Array(1024)], '01.in')
    const large = new File([new Uint8Array(16 * 1024 * 1024)], 'tests.zip')
    expect(totalTestFileSize([small, large])).toBe(small.size + large.size)
    expect(shouldUseChunkedUpload([small])).toBe(false)
    expect(shouldUseChunkedUpload([large])).toBe(true)
    expect(formatFileSize(2 * 1024 * 1024)).toBe('2.0 MB')
  })
})
