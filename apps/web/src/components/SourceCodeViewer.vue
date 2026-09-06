<template>
  <div class="source-viewer" role="region" aria-label="提交源代码" tabindex="0">
    <div class="source-action"><el-button size="small" :disabled="!source" @click="copySource">复制代码</el-button></div>
    <pre class="source-lines" aria-hidden="true">{{ sourceLineNumbers(source) }}</pre>
    <pre class="source-code"><code>{{ source }}</code></pre>
  </div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { copyTextToClipboard } from '../features/clipboard'
import { sourceLineNumbers } from '../features/submissions/presentation'
const props = defineProps<{ source: string }>()
async function copySource() {
  try { await copyTextToClipboard(props.source); ElMessage.success('代码已复制') }
  catch { ElMessage.error('复制失败') }
}
</script>

<style scoped>
.source-viewer { display: grid; grid-template-columns: auto auto minmax(0, 1fr); align-items: start; min-width: 0; min-height: 320px; max-height: 520px; overflow: auto; padding: 16px 12px; border: 1px solid var(--border); border-radius: 10px; background: #fff; color: #1f2937; }
.source-action { position: sticky; top: 0; padding-right: 8px; background: #fff; }
.source-lines, .source-code { margin: 0; font: 14px/24px ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; tab-size: 4; white-space: pre; }
.source-lines { padding: 0 12px 0 4px; text-align: right; color: #64748b; user-select: none; border-right: 1px solid #22a06b; }
.source-code { padding-left: 14px; min-width: 0; overflow-x: auto; }
.source-code code { font: inherit; }
@media (max-width: 600px) {
  .source-viewer { min-height: 260px; padding: 12px 6px; }
  .source-lines { padding-right: 8px; }
  .source-code { padding-left: 8px; }
}
</style>
