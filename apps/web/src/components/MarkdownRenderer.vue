<template>
  <div class="markdown-body" v-html="rendered" @click="onMarkdownClick"></div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onBeforeUnmount, shallowRef, watch } from 'vue'
import { getStatementAsset } from '../api/modules/assets'
import { copyTextToClipboard } from '../features/clipboard'
import { markdown, collectAssetPaths } from '../features/markdown/renderer'
import { mapConcurrent } from '../features/async'
import 'katex/dist/katex.min.css'

const props = defineProps<{ source?: string | null; problemId?: number; assetUrls?: Record<string, string>; assetBase?: string }>()
const fetchedAssetUrls = shallowRef<Record<string, string>>({})
const tokens = computed(() => markdown.parse(props.source || '', {}))
const pathsKey = computed(() => JSON.stringify(collectAssetPaths(tokens.value).filter(path => !props.assetUrls?.[path]).sort()))
const rendered = computed(() => markdown.renderer.render(tokens.value, markdown.options, {
  assetUrls: { ...fetchedAssetUrls.value, ...props.assetUrls }
}))
const copyTimers = new Map<HTMLButtonElement, ReturnType<typeof setTimeout>>()
let disposed = false

watch([() => props.problemId, () => props.assetBase, pathsKey], ([problemId, assetBase, paths], _previous, onCleanup) => {
  const controller = new AbortController()
  const loaded: Record<string, string> = {}
  fetchedAssetUrls.value = {}
  onCleanup(() => {
    controller.abort()
    for (const url of Object.values(loaded)) URL.revokeObjectURL(url)
  })
  if (!problemId) return
  void mapConcurrent(JSON.parse(paths) as string[], 4, async (path) => {
    if (controller.signal.aborted) return
    try {
      const { data } = await getStatementAsset(problemId, path, assetBase, controller.signal)
      if (!controller.signal.aborted) loaded[path] = URL.createObjectURL(data)
    } catch { /* Unavailable assets stay hidden. */ }
  }).then(() => {
    if (!controller.signal.aborted) fetchedAssetUrls.value = { ...loaded }
  })
}, { immediate: true })

async function onMarkdownClick(event: MouseEvent) {
  const button = (event.target as HTMLElement | null)?.closest<HTMLButtonElement>('.markdown-copy-button')
  if (!button) return
  const code = button.closest('.markdown-code-block')?.querySelector('pre code')?.textContent || ''
  try {
    await copyTextToClipboard(code)
    if (disposed || !button.isConnected) return
    button.textContent = '已复制'
    clearTimeout(copyTimers.get(button))
    copyTimers.set(button, setTimeout(() => { button.textContent = '复制'; copyTimers.delete(button) }, 1200))
  } catch { if (!disposed) ElMessage.error('复制失败，请手动选择文本') }
}
onBeforeUnmount(() => {
  disposed = true
  for (const timer of copyTimers.values()) clearTimeout(timer)
  copyTimers.clear()
})
</script>
<style scoped>
.markdown-body {
  color: var(--text);
  line-height: 1.75;
  overflow-wrap: anywhere;
}

.markdown-body :deep(p),
.markdown-body :deep(ul),
.markdown-body :deep(ol),
.markdown-body :deep(pre),
.markdown-body :deep(table),
.markdown-body :deep(blockquote) {
  margin: 0 0 12px;
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3) {
  margin: 18px 0 10px;
  line-height: 1.3;
}

.markdown-body :deep(pre) {
  overflow: auto;
  padding: 12px;
  border-radius: 8px;
  background: #111827;
  color: #f9fafb;
}

.markdown-body :deep(.markdown-code-block) {
  overflow: hidden;
  margin: 0 0 12px;
  border-radius: 8px;
  background: #111827;
}

.markdown-body :deep(.markdown-code-toolbar) {
  display: flex;
  justify-content: flex-end;
  min-height: 30px;
  padding: 5px 8px;
  border-bottom: 1px solid rgba(191, 219, 254, 0.16);
  background: rgba(15, 23, 42, 0.96);
}

.markdown-body :deep(.markdown-copy-button) {
  padding: 4px 9px;
  color: #dbeafe;
  border: 1px solid rgba(191, 219, 254, 0.32);
  border-radius: 6px;
  background: rgba(15, 23, 42, 0.82);
  font-size: 12px;
  cursor: pointer;
}

.markdown-body :deep(.markdown-code-block pre) {
  margin: 0;
  border-radius: 0;
}

.markdown-body :deep(code) {
  padding: 2px 5px;
  border-radius: 5px;
  background: rgba(15, 23, 42, 0.08);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.markdown-body :deep(pre code) {
  padding: 0;
  background: transparent;
}

.markdown-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  padding: 8px;
  border: 1px solid var(--border);
  text-align: left;
}

.markdown-body :deep(blockquote) {
  padding: 8px 12px;
  border-left: 4px solid var(--accent);
  background: rgba(10, 94, 166, 0.06);
}

.markdown-body :deep(img) {
  display: block;
  max-width: 100%;
  max-height: 520px;
  margin: 12px 0;
  border-radius: 8px;
  object-fit: contain;
  border: 1px solid var(--border);
  background: var(--surface-strong);
}

.math-block {
  overflow-x: auto;
  margin: 12px 0;
}
</style>
