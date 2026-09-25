<template>
  <div class="attachment-picker">
    <input ref="input" data-ticket-files type="file" multiple class="hidden-input" aria-label="选择工单附件" @change="choose" />
    <div class="attachment-summary">
      <div><strong role="status">已选择 {{ count }} / {{ limit }} 个附件（含保留附件）</strong><p>{{ formatSize(totalSize) }} / 96 MiB · 单个最多 32 MiB</p></div>
      <div class="attachment-actions"><el-button :disabled="disabled || count >= limit" @click="input?.click()">选择附件</el-button><el-button :disabled="disabled" @click="visible = true">管理附件{{ count ? `（${count}）` : '' }}</el-button></div>
    </div>
    <p class="attachment-help">测试点较多时建议上传 ZIP，包内使用同名 .in / .out 配对（可放在 tests 目录）；ZIP 按一个附件计数，解压后总计最多 96 MiB。零散文件可分次选择，提交时一并上传。</p>
    <el-dialog v-model="visible" title="工单附件" width="min(780px, calc(100vw - 24px))" append-to-body destroy-on-close>
      <div class="attachment-dialog-toolbar"><span>{{ count }} 个文件 · {{ formatSize(totalSize) }}</span><div class="attachment-actions"><el-button :disabled="disabled || !modelValue.length" @click="$emit('update:modelValue', [])">清空待上传</el-button><el-button :disabled="disabled || count >= limit" @click="input?.click()">继续选择</el-button></div></div>
      <div class="file-grid">
        <div v-for="file in retained" :key="`saved-${file.id}`" class="file-chip retained-file"><span class="file-kind">已保存</span><span class="file-name" :title="file.name">{{ file.name }}</span><small>{{ formatSize(file.size) }}</small><el-button link type="danger" :aria-label="`移除 ${file.name}`" :disabled="disabled" @click="$emit('update:retained', retained.filter(f => f.id !== file.id))">移除</el-button></div>
        <div v-for="(file, index) in modelValue" :key="file.name" class="file-chip"><span class="file-kind">{{ /\.zip$/i.test(file.name) ? 'ZIP' : /\.(in|out)$/i.test(file.name) ? '测试点' : '文件' }}</span><span class="file-name" :title="file.name">{{ file.name }}</span><small>{{ formatSize(file.size) }}</small><el-button link type="danger" :aria-label="`移除 ${file.name}`" :disabled="disabled" @click="$emit('update:modelValue', modelValue.filter((_, i) => i !== index))">移除</el-button></div>
        <el-empty v-if="!count" description="尚未选择附件" />
      </div>
      <template #footer><el-button type="primary" @click="visible = false">完成</el-button></template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { TicketAttachment } from '../../api/client'
const props = defineProps<{ modelValue: File[]; retained: TicketAttachment[]; limit: number; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [File[]]; 'update:retained': [TicketAttachment[]] }>()
const input = ref<HTMLInputElement>(), visible = ref(false)
const count = computed(() => props.modelValue.length + props.retained.length)
const totalSize = computed(() => [...props.modelValue, ...props.retained].reduce((sum, file) => sum + file.size, 0))
function formatSize(size: number) { return size < 1024 * 1024 ? `${(size / 1024).toFixed(1)} KiB` : `${(size / 1024 / 1024).toFixed(1)} MiB` }
function choose(event: Event) {
  const target = event.target as HTMLInputElement, files = Array.from(target.files || [])
  target.value = ''
  if (!files.length || props.disabled) return
  if (count.value + files.length > props.limit) return void ElMessage.warning(`最多上传 ${props.limit} 个附件（含保留附件）`)
  if (files.some(file => !file.size || file.size > 32 * 1024 * 1024)) return void ElMessage.warning('附件须非空，单个不超过 32 MiB')
  if (totalSize.value + files.reduce((n, f) => n + f.size, 0) > 96 * 1024 * 1024) return void ElMessage.warning('附件总计不能超过 96 MiB')
  const names = [...props.retained, ...props.modelValue, ...files].map(file => file.name)
  if (new Set(names).size !== names.length) return void ElMessage.warning('请为附件使用不同文件名')
  emit('update:modelValue', [...props.modelValue, ...files])
}
</script>
<style scoped>
.attachment-picker { width: 100%; min-width: 0; }
.hidden-input { display: none; }
.attachment-summary, .attachment-actions, .attachment-dialog-toolbar { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; }
.attachment-summary { padding: 16px; border: 1px solid var(--border); border-radius: 8px; background: var(--surface); }
.attachment-summary p, .attachment-help { color: var(--muted); margin: 6px 0 0; font-size: 12px; line-height: 1.7; }
.attachment-actions .el-button { margin: 0; }
.file-grid { display: grid; gap: 8px; max-height: min(55dvh, 480px); overflow-y: auto; margin-top: 16px; }
.file-chip { display: grid; grid-template-columns: 50px minmax(0, 1fr) auto auto; gap: 10px; align-items: center; padding: 10px 12px; background: var(--app-bg); border: 1px solid var(--border); border-radius: 7px; }
.file-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-kind, small { color: var(--muted); font-size: 11px; }
@media (max-width: 480px) { .file-chip { grid-template-columns: 42px minmax(0, 1fr) auto; gap: 6px; padding: 8px; } .file-chip small { display: none; } }
</style>
