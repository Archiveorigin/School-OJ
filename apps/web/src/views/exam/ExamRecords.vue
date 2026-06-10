<template>
  <section class="panel records-panel">
    <div class="section-title">
      <h3>提交记录</h3>
      <el-button @click="emit('refresh-history')">刷新</el-button>
    </div>
    <el-table :data="pagedHistory" size="small">
      <el-table-column label="题号" width="80">
        <template #default="{ row }">{{ problemLabel(row.problem_id) }}</template>
      </el-table-column>
      <el-table-column label="题目" min-width="180">
        <template #default="{ row }">{{ problemTitle(row.problem_id) }}</template>
      </el-table-column>
      <el-table-column prop="language" label="语言" width="90" />
      <el-table-column label="状态" width="130">
        <template #default="{ row }"><StatusBadge :status="row.status" /></template>
      </el-table-column>
      <el-table-column label="错误点" width="90">
        <template #default="{ row }">{{ row.error_point || '-' }}</template>
      </el-table-column>
      <el-table-column label="时间" min-width="160">
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
    </el-table>
    <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="history.length" />
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Problem, Submission } from '../../api/client'
import ListPagination from '../../components/ListPagination.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { formatDateTime } from '../../features/time'

const props = defineProps<{
  detail: any
  history: Submission[]
}>()

const emit = defineEmits<{
  'refresh-history': []
}>()

const page = ref(1)
const pageSize = ref(10)
const pagedHistory = computed(() => props.history.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))

function problemTitle(problemID: number) {
  return props.detail?.problems?.find((entry: { problem: Problem }) => entry.problem.id === problemID)?.problem.title || '-'
}

function problemLabel(problemID: number) {
  const index = props.detail?.problems?.findIndex((item: { problem: Problem }) => item.problem.id === problemID) ?? -1
  const entry = props.detail?.problems?.[index] as { problem: Problem; label?: string } | undefined
  if (entry?.label?.trim()) return entry.label.trim()
  return index >= 0 ? defaultProblemLabel(index) : '-'
}

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

function clampPage() {
  const maxPage = Math.max(1, Math.ceil(props.history.length / pageSize.value))
  if (page.value > maxPage) page.value = maxPage
  if (page.value < 1) page.value = 1
}

watch(() => props.history.length, clampPage)
watch(pageSize, clampPage)
</script>

<style scoped>
.records-panel {
  min-width: 0;
}
</style>
