<template>
  <el-tag :type="type" effect="plain">{{ label }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const type = computed(() => {
  if (props.status === 'accepted' || props.status === 'manual_graded') return 'success'
  if (props.status === 'queued' || props.status === 'running' || props.status === 'pending_review') return 'warning'
  if (props.status === 'system_error') return 'danger'
  return 'info'
})

const label = computed(() => {
  const map: Record<string, string> = {
    queued: '排队中',
    running: '判题中',
    pending_review: '待评分',
    manual_graded: '已评分',
    accepted: '通过',
    wrong_answer: '答案错误',
    compile_error: '编译错误',
    runtime_error: '运行错误',
    time_limit: '超时',
    memory_limit: '超内存',
    output_limit: '输出超限',
    system_error: '系统错误'
  }
  return map[props.status] || props.status
})
</script>
