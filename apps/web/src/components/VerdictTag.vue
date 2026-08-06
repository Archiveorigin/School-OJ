<template>
  <el-tag :type="tagType" effect="dark" class="verdict-tag">{{ code }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const code = computed(() => ({
  accepted: 'AC',
  wrong_answer: 'WA',
  runtime_error: 'RE',
  time_limit: 'TLE',
  memory_limit: 'MLE',
  output_limit: 'OLE',
  compile_error: 'CE',
  system_error: 'SE',
  queued: 'QUEUE',
  running: 'JUDGING',
  pending_review: 'REVIEW',
  manual_graded: 'GRADED'
} as Record<string, string>)[props.status] || props.status.toUpperCase())

const tagType = computed<'success' | 'warning' | 'danger' | 'info'>(() => {
  if (props.status === 'accepted' || props.status === 'manual_graded') return 'success'
  if (['queued', 'running', 'pending_review'].includes(props.status)) return 'warning'
  if (props.status) return 'danger'
  return 'info'
})
</script>

<style scoped>
.verdict-tag { min-width: 44px; justify-content: center; font-weight: 900; letter-spacing: .04em; }
</style>
