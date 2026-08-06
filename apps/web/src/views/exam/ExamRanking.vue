<template>
  <section class="ranking-view">
    <div class="ranking-toolbar">
      <div>
        <span class="eyebrow">LIVE SCOREBOARD</span>
        <h3>考试实时榜单</h3>
      </div>
      <div class="toolbar">
        <el-switch v-model="autoRefresh" active-text="自动刷新" />
        <span class="muted">{{ lastLoadedAt ? `更新于 ${formatDateTime(lastLoadedAt)}` : '' }}</span>
        <el-button :loading="loading" @click="loadRanking">刷新</el-button>
      </div>
    </div>

    <div v-if="error" class="empty-ranking">
      <strong>榜单暂不可用</strong><span>{{ error }}</span>
    </div>
    <div v-else-if="ranking && !ranking.has_class" class="empty-ranking">
      <strong>该考试未绑定班级</strong><span>实时榜需要按班级学生生成。</span>
    </div>
    <el-table v-else :data="ranking?.rows || []" v-loading="loading" max-height="620">
      <el-table-column prop="rank" label="排名" width="76" fixed />
      <el-table-column label="学生" min-width="170" fixed>
        <template #default="{ row }"><div class="student-cell"><strong>{{ row.name }}</strong><span>{{ row.student_no || '-' }}</span></div></template>
      </el-table-column>
      <el-table-column label="总分" width="116" fixed>
        <template #default="{ row }"><strong>{{ row.total_score }} / {{ row.max_score }}</strong></template>
      </el-table-column>
      <el-table-column v-for="problem in ranking?.problems || []" :key="problem.problem_id" :label="problem.label || problem.display_code" width="118" align="center">
        <template #default="{ row }">
          <div class="problem-score"><strong>{{ scoreText(problemCell(row, problem.problem_id)) }}</strong><small>{{ statusText(problemCell(row, problem.problem_id)) }}</small></div>
        </template>
      </el-table-column>
      <el-table-column prop="solved" label="通过" width="78" />
      <el-table-column prop="submission_count" label="提交" width="78" />
      <el-table-column label="最后提交" min-width="170"><template #default="{ row }">{{ formatDateTime(row.last_submission) }}</template></el-table-column>
    </el-table>
  </section>
</template>

<script setup lang="ts">
defineOptions({ inheritAttrs: false })

import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { client } from '../../api/client'
import { formatDateTime } from '../../features/time'

const route = useRoute()
const ranking = ref<any>(null)
const loading = ref(false)
const autoRefresh = ref(true)
const error = ref('')
const lastLoadedAt = ref<Date | null>(null)
let refreshTimer: number | undefined

async function loadRanking() {
  const examID = Number(route.params.id)
  if (!examID) return
  loading.value = true
  error.value = ''
  try {
    ranking.value = (await client.get(`/exams/${examID}/ranking`)).data
    lastLoadedAt.value = new Date()
  } catch (err: any) {
    ranking.value = null
    error.value = err.response?.status === 403 ? '创建者未向考生开放该榜单。' : (err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

function resetTimer() {
  if (refreshTimer) window.clearInterval(refreshTimer)
  refreshTimer = undefined
  if (autoRefresh.value) refreshTimer = window.setInterval(loadRanking, 5000)
}

function problemCell(row: any, problemID: number) {
  return row.problems?.find((item: any) => item.problem_id === problemID)
}

function scoreText(cell: any) {
  if (!cell) return '-'
  if (cell.pending) return '待评分'
  return cell.score_ready ? `${cell.best_score} / ${cell.max_score}` : '-'
}

function statusText(cell: any) {
  if (!cell?.status) return '未提交'
  return cell.pending ? '评分中' : String(cell.status).replace(/_/g, ' ')
}

watch(autoRefresh, resetTimer)
onMounted(() => { loadRanking(); resetTimer() })
onBeforeUnmount(() => { if (refreshTimer) window.clearInterval(refreshTimer) })
</script>

<style scoped>
.ranking-view { display: grid; gap: 16px; padding: 2px; }
.ranking-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 18px; }
.ranking-toolbar h3 { margin: 5px 0 0; }
.eyebrow { color: var(--accent); font-size: 10px; font-weight: 800; letter-spacing: .14em; }
.student-cell span, .problem-score small, .empty-ranking span { color: var(--muted); font-size: 12px; }
.student-cell, .problem-score, .empty-ranking { display: grid; gap: 3px; }
.empty-ranking { place-items: center; padding: 52px 20px; border: 1px dashed var(--border); border-radius: 12px; }
@media (max-width: 760px) { .ranking-toolbar { align-items: stretch; flex-direction: column; } }
</style>
