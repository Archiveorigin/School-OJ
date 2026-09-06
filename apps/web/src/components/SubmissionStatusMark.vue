<template>
  <svg
    class="submission-status-mark"
    xmlns="http://www.w3.org/2000/svg"
    :viewBox="compact ? '0 0 64 28' : '0 0 156 48'"
    :width="compact ? 64 : 156"
    :height="compact ? 28 : 48"
    fill="none"
    role="img"
    :aria-label="visual.ariaLabel"
  >
    <title>{{ visual.ariaLabel }}</title>
    <rect :width="compact ? 64 : 156" :height="compact ? 28 : 48" :rx="compact ? 6 : 14" :fill="visual.fill" />
    <text
      :x="compact ? 32 : 78"
      :y="compact ? 19 : 30"
      fill="white"
      text-anchor="middle"
      :font-size="compact ? 13 : 17"
      font-weight="800"
      font-family="Inter, Microsoft YaHei, Arial, sans-serif"
    >
      {{ compact ? shortVerdicts[status] || visual.label : visual.label }}
    </text>
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { shortVerdicts } from '../features/submissions/presentation'

const props = defineProps<{ status: string; compact?: boolean }>()

const terminalFailures = new Set([
  'wrong_answer',
  'compile_error',
  'runtime_error',
  'time_limit',
  'memory_limit',
  'output_limit',
  'system_error',
])

const visual = computed(() => {
  const status = props.status.trim().toLowerCase()

  if (status === 'recorded') return { label: '已记录', ariaLabel: '已记录', fill: '#64748b' }

  if (status === 'accepted' || status === 'ac') {
    return { label: 'Accepted', ariaLabel: '通过', fill: '#22c55e' }
  }

  if (status === 'uac' || terminalFailures.has(status)) {
    return { label: 'Unaccepted', ariaLabel: '未通过', fill: '#ef4444' }
  }

  if (status === 'queued') {
    return { label: '排队中', ariaLabel: '排队中', fill: '#3b82f6' }
  }

  if (status === 'pending_review') {
    return { label: '待评阅', ariaLabel: '待评阅', fill: '#d97706' }
  }

  if (status === 'manual_graded') {
    return { label: '已评分', ariaLabel: '已评分', fill: '#d97706' }
  }

  return { label: '评测中', ariaLabel: '评测中', fill: '#3b82f6' }
})
</script>

<style scoped>
.submission-status-mark {
  display: block;
  max-width: 100%;
  height: auto;
  flex: 0 0 auto;
}
</style>
