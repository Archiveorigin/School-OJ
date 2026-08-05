<template>
  <template v-if="activeProblem">
    <ProblemStatementView
      :problem="activeProblem"
      :problem-number="displayNumber"
      :score="activeEntry?.score"
      :status-text="statusText"
      :status-type="statusType"
      :status-image="statusImage"
      :show-difficulty="false"
      :show-meta="false"
    />
  </template>
  <div v-else class="panel empty-detail muted">请选择题目</div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Problem } from '../../api/client'
import ProblemStatementView from '../../components/ProblemStatementView.vue'

const props = defineProps<{
  detail: any
  activeEntry: {
    problem: Problem
    score: number
    label?: string
    problem_id: number
  } | null
  activeProblem: Problem | null
  canManage?: boolean
  switcherEntries: Array<{ problem: Problem; label?: string; submission_status?: string }>
  activeProblemId?: number
}>()

const scoreItem = computed(() => {
  return props.detail?.problem_scores?.find((item: any) => item.problem.id === props.activeProblem?.id)
})
const statusText = computed(() => {
  const item = scoreItem.value
  if (!item?.submission_id) return '未提交'
  if (props.detail?.manual_review && !item.score_ready) return '待评分'
  if (!item.score_ready) return '计算中'
  return `${item.best_score} / ${item.score}`
})
const statusType = computed<'success' | 'warning' | 'info' | 'danger'>(() => {
  const item = scoreItem.value
  if (!item?.submission_id) return 'info'
  if (!item.score_ready) return 'warning'
  if (item.best_score >= item.score) return 'success'
  if (item.best_score > 0) return 'warning'
  return 'danger'
})
const statusImage = computed<'ac' | 'uac' | ''>(() => {
  const item = scoreItem.value
  if (!item?.submission_id || !item.score_ready) return ''
  return item.best_score >= item.score ? 'ac' : 'uac'
})
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
.empty-detail {
  display: grid;
  min-height: 260px;
  place-items: center;
}

</style>
