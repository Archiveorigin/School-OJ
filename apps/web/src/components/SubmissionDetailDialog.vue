<template>
  <el-dialog v-model="visible" title="提交代码" width="min(1200px, calc(100vw - 28px))">
    <div v-if="submission" class="submission-detail-body">
      <div class="submission-detail-summary">
        <article>
          <span>提交人</span>
          <strong>{{ submitterText }}</strong>
        </article>
        <article>
          <span>语言</span>
          <strong>{{ languageLabel(submission.language) }}</strong>
        </article>
        <article>
          <span>状态</span>
          <SubmissionStatusMark :status="submission.status" class="submission-detail-status" />
        </article>
        <article>
          <span>提交时间</span>
          <strong>{{ formatDateTime(submission.created_at) }}</strong>
        </article>
      </div>

      <div class="submission-source-heading">
        <strong>源代码</strong>
        <el-button size="small" :disabled="!submission.source_code" @click="copySource">复制代码</el-button>
      </div>
      <pre class="submission-source-code">{{ submission.source_code || '' }}</pre>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed } from 'vue'
import type { Submission } from '../api/client'
import { copyTextToClipboard } from '../features/clipboard'
import { formatDateTime } from '../features/time'
import SubmissionStatusMark from './SubmissionStatusMark.vue'

const props = defineProps<{
  modelValue: boolean
  detail: { submission: Submission } | null
}>()

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})
const submission = computed(() => props.detail?.submission || null)
const submitterText = computed(() => {
  const item = submission.value
  if (!item?.user_name) return '-'
  return item.user_name
})

function languageLabel(language: string) {
  return ({ cpp: 'C++17', c: 'C', python: 'Python 3', java: 'Java 21' } as Record<string, string>)[language] || language
}

async function copySource() {
  try {
    await copyTextToClipboard(submission.value?.source_code || '')
    ElMessage.success('代码已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}
</script>

<style scoped>
.submission-detail-body { display: grid; gap: 14px; }
.submission-detail-summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
.submission-detail-summary article { display: grid; align-content: start; gap: 7px; min-width: 0; padding: 16px; border: 1px solid #d5deec; border-radius: 11px; }
.submission-detail-summary span { color: var(--muted); font-size: 12px; }
.submission-detail-summary strong { overflow-wrap: anywhere; }
.submission-detail-status { width: min(156px, 100%); }
.submission-source-heading { display: flex; align-items: center; justify-content: space-between; }
.submission-source-code { width: 100%; max-width: 100%; max-height: 520px; min-height: 350px; margin: 0; padding: 16px; overflow: auto; color: #e2e8f0; border-radius: 12px; background: #0f172a; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; white-space: pre; }
@media (max-width: 760px) {
  .submission-detail-summary { grid-template-columns: 1fr; }
  .submission-source-code { min-height: 260px; }
}
</style>
