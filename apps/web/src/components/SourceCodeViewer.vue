<template>
  <div class="source-viewer" role="region" aria-label="提交源代码" tabindex="0">
    <div class="source-scroll">
      <pre class="source-lines" aria-hidden="true">{{ lineNumbers }}</pre>
      <pre class="source-code"><code>{{ source }}</code></pre>
    </div>
    <div class="source-action"><el-button size="small" :disabled="!source" @click="copySource">复制代码</el-button></div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElMessage } from 'element-plus'
import { copyTextToClipboard } from '../features/clipboard'
import { sourceLineNumbers } from '../features/submissions/presentation'
const props = defineProps<{ source: string }>()
const lineNumbers = computed(() => sourceLineNumbers(props.source))
async function copySource() {
  try { await copyTextToClipboard(props.source); ElMessage.success('代码已复制') }
  catch { ElMessage.error('复制失败') }
}
</script>

<style scoped>
.source-viewer { position: relative; min-width: 0; overflow: hidden; border: 1px solid var(--color-border); border-radius: var(--radius-md); background: var(--code-surface); color: var(--code-text); }
.source-action { position: absolute; top: 10px; right: 12px; z-index: 1; display: flex; justify-content: flex-end; opacity: 0; pointer-events: none; transition: opacity 120ms ease; }
.source-viewer:hover .source-action, .source-viewer:has(:focus-visible) .source-action, .source-viewer:focus-visible .source-action { opacity: 1; pointer-events: auto; }
.source-scroll { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: start; min-height: 270px; max-height: 470px; overflow: auto; padding: 44px 12px 16px; }
.source-lines, .source-code { margin: 0; font: 14px/24px ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; tab-size: 4; white-space: pre; }
.source-lines { padding: 0 12px 0 4px; text-align: right; color: var(--code-line-number); user-select: none; border-right: 1px solid var(--code-divider); }
.source-code { padding-left: 14px; min-width: 0; overflow-x: auto; }
.source-code code { font: inherit; }
@media (hover: none) {
  .source-action { opacity: 1; pointer-events: auto; }
}
@media (max-width: 600px) {
  .source-scroll { min-height: 210px; padding-inline: 6px; }
  .source-lines { padding-right: 8px; }
  .source-code { padding-left: 8px; }
}
</style>
