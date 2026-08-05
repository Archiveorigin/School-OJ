import { client } from '../../api/client'

export const TEST_UPLOAD_CHUNK_SIZE = 4 * 1024 * 1024
export const TEST_UPLOAD_CHUNK_THRESHOLD = 16 * 1024 * 1024
export const MAX_TEST_UPLOAD_SIZE = 128 * 1024 * 1024

export type TestUploadReference = {
  id: string
  name: string
  chunk_count: number
  size: number
}

type ChunkTask = {
  key: string
  uploadID: string
  chunkIndex: number
  start: number
  end: number
  file: File
}

export function totalTestFileSize(files: File[]) {
  return files.reduce((total, file) => total + file.size, 0)
}

export function shouldUseChunkedUpload(files: File[]) {
  return totalTestFileSize(files) >= TEST_UPLOAD_CHUNK_THRESHOLD
}

export function formatFileSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

export function buildChunkPlan(files: File[]) {
  const references: TestUploadReference[] = []
  const tasks: ChunkTask[] = []
  files.forEach((file, fileIndex) => {
    const uploadID = newUploadID()
    const chunkCount = Math.ceil(file.size / TEST_UPLOAD_CHUNK_SIZE)
    references.push({ id: uploadID, name: file.name, chunk_count: chunkCount, size: file.size })
    for (let chunkIndex = 0; chunkIndex < chunkCount; chunkIndex += 1) {
      const start = chunkIndex * TEST_UPLOAD_CHUNK_SIZE
      tasks.push({
        key: `${fileIndex}:${chunkIndex}`,
        uploadID,
        chunkIndex,
        start,
        end: Math.min(file.size, start + TEST_UPLOAD_CHUNK_SIZE),
        file
      })
    }
  })
  return { references, tasks }
}

export async function uploadTestFilesInChunks(files: File[], onProgress: (percent: number) => void) {
  const totalSize = totalTestFileSize(files)
  if (!totalSize || totalSize > MAX_TEST_UPLOAD_SIZE) throw new Error('测试点文件总大小不能超过 128MB')
  const { references, tasks } = buildChunkPlan(files)
  const loadedByTask = new Map<string, number>()
  let cursor = 0
  const report = () => {
    const loaded = [...loadedByTask.values()].reduce((sum, value) => sum + value, 0)
    onProgress(Math.min(100, Math.round((loaded / totalSize) * 100)))
  }
  const worker = async () => {
    while (cursor < tasks.length) {
      const task = tasks[cursor]
      cursor += 1
      const size = task.end - task.start
      await client.put(`/problem-test-uploads/${task.uploadID}/${task.chunkIndex}`, task.file.slice(task.start, task.end), {
        headers: { 'Content-Type': 'application/octet-stream' },
        timeout: 120_000,
        onUploadProgress(event) {
          loadedByTask.set(task.key, Math.min(size, event.loaded))
          report()
        }
      })
      loadedByTask.set(task.key, size)
      report()
    }
  }
  await Promise.all(Array.from({ length: Math.min(3, tasks.length) }, () => worker()))
  onProgress(100)
  return references
}

function newUploadID() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') return crypto.randomUUID()
  if (typeof crypto === 'undefined') return `${Date.now().toString(16)}-${Math.random().toString(16).slice(2)}-${Math.random().toString(16).slice(2)}`
  const values = new Uint32Array(4)
  crypto.getRandomValues(values)
  return [...values].map((value) => value.toString(16).padStart(8, '0')).join('-')
}
