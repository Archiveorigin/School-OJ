<template>
  <ProblemOverview embedded :items="switcherEntries" :active-problem-id="activeProblemId" @select="openProblem" />
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import type { Problem } from '../../api/client'
import ProblemOverview from '../../components/ProblemOverview.vue'

defineProps<{
  switcherEntries: Array<{ problem: Problem; label?: string; score?: number; submission_status?: string }>
  activeProblemId?: number
}>()
const emit = defineEmits<{ 'update:active-problem-id': [value: number] }>()
const route = useRoute()
const router = useRouter()

function openProblem(problemID: number) {
  emit('update:active-problem-id', problemID)
  router.push(`/exams/${route.params.id}/problems`)
}
</script>
