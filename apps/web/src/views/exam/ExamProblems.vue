<template>
  <ProblemStatementView
    v-if="activeProblem"
    :problem="activeProblem"
    :problem-number="activeEntry?.label || activeProblem.id"
    :score="activeEntry?.score"
    :status-text="statusText"
    :status-type="statusType"
    :show-difficulty="false"
  />
  <div v-else class="panel empty-detail muted">请选择题目</div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Problem } from '../../api/client'
import ProblemStatementView from '../../components/ProblemStatementView.vue'

const props = defineProps<{
  detail: any
  activeEntry: { problem: Problem; score: number; label?: string; problem_id: number } | null
  activeProblem: Problem | null
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
</script>

<style scoped>
.empty-detail {
  display: grid;
  min-height: 260px;
  place-items: center;
}
</style>
