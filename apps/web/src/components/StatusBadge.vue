<template>
  <el-tag :type="type" effect="plain">{{ status }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const type = computed(() => {
  if (props.status === 'accepted') return 'success'
  if (props.status === 'queued' || props.status === 'running') return 'warning'
  if (props.status === 'system_error') return 'danger'
  return 'info'
})
</script>
