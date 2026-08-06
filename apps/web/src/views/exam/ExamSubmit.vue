<template>
  <section v-if="activeProblem" class="panel submit-panel">
    <div class="section-title">
      <h3>提交代码</h3>
      <span class="muted">{{ displayNumber }} {{ activeProblem.title }}</span>
    </div>
    <SubmissionComposer
      :key="activeProblem.id"
      :language="language"
      :source="source"
      :status="currentStatus"
      :message="currentMessage"
      :submitting="submitting"
      :disabled="!detail.can_submit"
      :draft-context="draftContext"
      scope-text="本次代码仅计入当前考试"
      @update:language="emit('update:language', $event)"
      @update:source="emit('update:source', $event)"
      @submit="emit('submit')"
    />
  </section>
  <div v-else class="panel empty-detail muted">请选择题目</div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Problem, Submission } from '../../api/client'
import SubmissionComposer from '../../components/SubmissionComposer.vue'

const props = defineProps<{
  detail: any
  activeEntry: { problem: Problem; score: number; label?: string; problem_id: number } | null
  activeProblem: Problem | null
  language: string
  source: string
  live: any
  history?: Submission[]
  latestHistory?: Submission[]
  submitting: boolean
  switcherEntries: Array<{ problem: Problem; label?: string; submission_status?: string }>
  activeProblemId?: number
  userId: number
}>()

const emit = defineEmits<{
  'update:language': [value: string]
  'update:source': [value: string]
  submit: []
  'update:active-problem-id': [value: number]
}>()

const latestSubmission = computed(() => props.latestHistory?.find((item) => item.problem_id === props.activeProblem?.id))
const draftContext = computed(() => ({ userId: props.userId, resourceType: 'exam' as const, resourceId: Number(props.detail?.exam?.id), problemId: props.activeProblem?.id || 0 }))
const currentStatus = computed(() => props.live?.status || latestSubmission.value?.status || '')
const currentMessage = computed(() => props.live?.message || latestSubmission.value?.message || '')
const displayNumber = computed(() => {
  if (props.activeEntry?.label?.trim()) return props.activeEntry.label.trim()
  const index = props.detail?.problems?.findIndex((entry: { problem: Problem }) => entry.problem.id === props.activeProblem?.id) ?? 0
  return defaultProblemLabel(index >= 0 ? index : 0)
})

function defaultProblemLabel(index: number) {
  index += 1
  let label = ''
  while (index > 0) {
    index -= 1
    label = String.fromCharCode(65 + (index % 26)) + label
    index = Math.floor(index / 26)
  }
  return label
}
</script>

<style scoped>
.submit-panel {
  display: grid;
  gap: 10px;
}

.empty-detail {
  display: grid;
  min-height: 260px;
  place-items: center;
}

</style>
