<template>
  <el-dialog v-model="visible" title="提交详情" width="min(1200px, calc(100vw - 28px))">
    <div v-if="submission" class="submission-detail-body">
      <div class="submission-detail-summary">
        <span class="result">提交结果：<SubmissionStatusMark :status="submission.status" compact /></span>
        <span>运行时间：{{ showMetrics ? `${submission.time_ms}ms` : '—' }}</span>
        <span>占用内存：{{ showMetrics ? `${submission.memory_kb}KB` : '—' }}</span>
        <span>使用语言：{{ languageLabel(submission.language) }}</span>
        <span>用例通过情况：{{ caseSummary }}</span>
      </div>

      <SourceCodeViewer :source="submission.source_code || ''" />
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Submission } from '../api/client'
import { hasEvaluatedResources, languageLabel } from '../features/submissions/presentation'
import SourceCodeViewer from './SourceCodeViewer.vue'
import SubmissionStatusMark from './SubmissionStatusMark.vue'

const props = defineProps<{
  modelValue: boolean
  detail: { submission: Submission; case_summary?: { passed: number; total: number } | null } | null
}>()

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})
const submission = computed(() => props.detail?.submission || null)
const showMetrics = computed(() => hasEvaluatedResources(submission.value?.status || ''))
const caseSummary = computed(() => {
  const summary = props.detail?.case_summary
  return showMetrics.value && summary ? `${summary.passed}/${summary.total}` : '—'
})
</script>

<style scoped>
.submission-detail-body { display: grid; gap: 20px; min-width: 0; }
.submission-detail-summary { display: flex; flex-wrap: wrap; align-items: center; gap: 12px 24px; color: var(--text); line-height: 1.7; }
.submission-detail-summary > span { white-space: nowrap; }
.result { display: inline-flex; align-items: center; gap: 4px; }
</style>
