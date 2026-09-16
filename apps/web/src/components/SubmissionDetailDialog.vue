<template>
  <el-dialog v-model="visible" title="提交详情" width="min(1200px, calc(100vw - 28px))" append-to-body destroy-on-close>
    <SubmissionDetailContent :detail="detail" />
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Submission } from '../api/client'
import SubmissionDetailContent from './SubmissionDetailContent.vue'

const props = defineProps<{
  modelValue: boolean
  detail: { submission: Submission; case_summary?: { passed: number; total: number } | null } | null
}>()

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})
</script>
