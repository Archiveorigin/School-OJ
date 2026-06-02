<template>
  <section class="panel records-panel">
    <div class="section-title">
      <h3>提交记录</h3>
      <el-button @click="emit('refresh-history')">刷新</el-button>
    </div>
    <el-table :data="history" size="small">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column label="题目" min-width="180">
        <template #default="{ row }">{{ problemTitle(row.problem_id) }}</template>
      </el-table-column>
      <el-table-column prop="language" label="语言" width="90" />
      <el-table-column label="状态" width="130">
        <template #default="{ row }"><StatusBadge :status="row.status" /></template>
      </el-table-column>
      <el-table-column prop="score" label="参考分" width="90" />
      <el-table-column label="最终分" width="90">
        <template #default="{ row }">{{ row.manual_score ?? '-' }}</template>
      </el-table-column>
      <el-table-column label="时间" min-width="160">
        <template #default="{ row }">{{ row.created_at }}</template>
      </el-table-column>
    </el-table>
  </section>
</template>

<script setup lang="ts">
import type { Problem, Submission } from '../../api/client'
import StatusBadge from '../../components/StatusBadge.vue'

const props = defineProps<{
  detail: any
  history: Submission[]
}>()

const emit = defineEmits<{
  'refresh-history': []
}>()

function problemTitle(problemID: number) {
  return props.detail?.problems?.find((entry: { problem: Problem }) => entry.problem.id === problemID)?.problem.title || `#${problemID}`
}
</script>

<style scoped>
.records-panel {
  min-width: 0;
}
</style>
