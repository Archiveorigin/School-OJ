import { generatedApi } from "../../api/client";
import { randomUUID } from "../uuid";
import { eventPath, type EventKind } from "../../api/modules/events";
import {
  type TestUploadReference,
  totalTestFileSize,
} from "../problems/testFileUpload";
export const EVENT_CHUNK_SIZE = 1024 * 1024;
export const TEST_PREVIEW_BYTES = 64 * 1024;
export function testFileError(files: File[]) {
  if (files.length > 400 || totalTestFileSize(files) > 128 * 1024 * 1024)
    return "最多 400 个测试文件，总计 128 MiB";
  const names = new Set<string>();
  for (const f of files) {
    const name = f.name.toLowerCase();
    if (!/\.(in|out|zip)$/.test(name)) return "请选择 .in / .out 文件或 ZIP";
    if (names.has(name)) return `文件重名：${f.name}`;
    names.add(name);
  }
  for (const f of files) {
    if (
      /\.in$/i.test(f.name) &&
      !names.has(f.name.replace(/\.in$/i, ".out").toLowerCase())
    )
      return `缺少 ${f.name} 的输出文件`;
    if (
      /\.(out)$/i.test(f.name) &&
      !names.has(f.name.replace(/\.(out)$/i, ".in").toLowerCase())
    )
      return `缺少 ${f.name} 的输入文件`;
  }
  return "";
}
export async function previewTestFile(file: File) {
  return {
    text: await file.slice(0, TEST_PREVIEW_BYTES).text(),
    truncated: file.size > TEST_PREVIEW_BYTES,
  };
}
export async function uploadEventTests(
  files: File[],
  kind: EventKind,
  eventId: number,
  revision: number,
  progress: (value: number) => void,
  signal: AbortSignal,
) {
  const error = testFileError(files);
  if (error) throw new Error(error);
  const refs: TestUploadReference[] = [],
    tasks: Array<{
      file: File;
      id: string;
      index: number;
      start: number;
      end: number;
    }> = [];
  for (const file of files) {
    const id = randomUUID(),
      count = Math.max(1, Math.ceil(file.size / EVENT_CHUNK_SIZE));
    refs.push({ id, name: file.name, size: file.size, chunk_count: count });
    for (let index = 0; index < count; index++)
      tasks.push({
        file,
        id,
        index,
        start: index * EVENT_CHUNK_SIZE,
        end: Math.min(file.size, (index + 1) * EVENT_CHUNK_SIZE),
      });
  }
  const loaded = new Map<number, number>(),
    total = totalTestFileSize(files) || 1;
  let cursor = 0;
  const worker = async () => {
    while (cursor < tasks.length) {
      signal.throwIfAborted();
      const key = cursor++,
        task = tasks[key];
      for (let retry = 0; ; retry++) {
        try {
          await generatedApi.uploadEventTestChunk({
            path: {
              ...eventPath(kind, eventId),
              upload_id: task.id,
              chunk_index: task.index,
            },
            query: { revision },
            body: task.file.slice(task.start, task.end),
            headers: { "Content-Type": "application/octet-stream" },
            signal,
            timeout: 120000,
            onUploadProgress(e) {
              loaded.set(key, Math.min(e.loaded, task.end - task.start));
              progress(
                Math.min(
                  99,
                  Math.floor(
                    ([...loaded.values()].reduce((a, b) => a + b, 0) / total) *
                      100,
                  ),
                ),
              );
            },
          });
          break;
        } catch (e: any) {
          if (
            signal.aborted ||
            retry >= 1 ||
            (e.response?.status && e.response.status < 500)
          )
            throw e;
        }
      }
      loaded.set(key, task.end - task.start);
    }
  };
  await Promise.all(Array.from({ length: Math.min(3, tasks.length) }, worker));
  progress(100);
  return refs;
}
