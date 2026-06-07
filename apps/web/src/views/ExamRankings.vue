<template>
  <section class="page exam-ranking-page">
    <div class="page-header">
      <div>
        <h2>考试实时榜</h2>
        <p class="muted">
          {{ ranking?.exam?.course_code || selectedExam?.course_code || '-' }} /
          {{ ranking?.exam?.class_name || selectedExam?.class_name || '-' }}
        </p>
      </div>
      <div class="toolbar">
        <el-select v-model="selectedExamID" filterable placeholder="选择考试" class="exam-select" @change="loadRanking">
          <el-option
            v-for="exam in exams"
            :key="exam.id"
            :label="examOptionLabel(exam)"
            :value="exam.id"
          />
        </el-select>
        <el-switch v-model="autoRefresh" active-text="自动刷新" />
        <el-button :loading="loading" @click="loadRanking">刷新</el-button>
      </div>
    </div>

    <div class="scoreboard-hero panel">
      <div>
        <span class="eyebrow">{{ ranking?.exam?.status || examStatus(selectedExam) }}</span>
        <h3>{{ ranking?.exam?.title || selectedExam?.title || '暂无考试' }}</h3>
        <p class="muted">
          {{ formatDateTime(ranking?.exam?.starts_at || selectedExam?.starts_at) }}
          -
          {{ formatDateTime(ranking?.exam?.ends_at || selectedExam?.ends_at) }}
        </p>
      </div>
      <div class="ranking-stats">
        <div>
          <strong>{{ ranking?.stats?.total_students || 0 }}</strong>
          <span>学生</span>
        </div>
        <div>
          <strong>{{ ranking?.stats?.finished || 0 }}</strong>
          <span>已结束</span>
        </div>
        <div>
          <strong>{{ ranking?.stats?.pending || 0 }}</strong>
          <span>待评分</span>
        </div>
        <div>
          <strong>{{ ranking?.stats?.max_score || 0 }}</strong>
          <span>总分</span>
        </div>
      </div>
    </div>

    <div v-if="ranking && !ranking.has_class" class="panel empty-state">
      <strong>该考试未绑定班级</strong>
      <span class="muted">实时榜需要按班级学生生成。</span>
    </div>
    <div v-else-if="!selectedExamID" class="panel empty-state">
      <strong>暂无可展示的考试</strong>
      <span class="muted">创建考试后会在这里生成实时榜。</span>
    </div>
    <div v-else class="panel ranking-panel">
      <div class="section-title">
        <h3>实时排名</h3>
        <span class="muted">最后刷新：{{ formatDateTime(lastLoadedAt) }}</span>
      </div>
      <el-table :data="ranking?.rows || []" :default-sort="{ prop: 'rank', order: 'ascending' }" height="calc(100vh - 360px)">
        <el-table-column prop="rank" label="排名" width="82" fixed />
        <el-table-column label="学生" min-width="180" fixed>
          <template #default="{ row }">
            <div class="student-cell">
              <strong>{{ row.name }}</strong>
              <span>{{ row.student_no || '-' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="总分" width="110" fixed>
          <template #default="{ row }">
            <strong>{{ row.total_score }} / {{ row.max_score }}</strong>
          </template>
        </el-table-column>
        <el-table-column
          v-for="problem in ranking?.problems || []"
          :key="problem.problem_id"
          :label="problem.label || problem.display_code"
          width="116"
          align="center"
        >
          <template #default="{ row }">
            <div class="problem-score-cell">
              <strong>{{ scoreText(problemCell(row, problem.problem_id)) }}</strong>
              <small>{{ statusText(problemCell(row, problem.problem_id)) }}</small>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="solved" label="通过" width="82" />
        <el-table-column prop="attempted" label="尝试" width="82" />
        <el-table-column prop="submission_count" label="提交" width="82" />
        <el-table-column label="最后提交" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.last_submission) }}</template>
        </el-table-column>
        <el-table-column label="结束考试" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.finished_at) }}</template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { client } from '../api/client'
import { formatDateTime } from '../features/time'

const exams = ref<any[]>([])
const selectedExamID = ref<number>()
const ranking = ref<any>(null)
const loading = ref(false)
const autoRefresh = ref(true)
const lastLoadedAt = ref<Date | null>(null)
let refreshTimer: number | undefined

const selectedExam = computed(() => exams.value.find((exam) => exam.id === selectedExamID.value))

async function loadExams() {
  const { data } = await client.get('/exams')
  exams.value = Array.isArray(data) ? data : []
  if (!selectedExamID.value && exams.value.length) {
    selectedExamID.value = preferredExamID()
  }
}

function preferredExamID() {
  const now = Date.now()
  const active = exams.value.find((exam) => {
    const start = exam.starts_at ? new Date(exam.starts_at).getTime() : 0
    const end = exam.ends_at ? new Date(exam.ends_at).getTime() : Number.POSITIVE_INFINITY
    return start <= now && now < end
  })
  return active?.id || exams.value[0]?.id
}

async function loadRanking() {
  if (!selectedExamID.value) {
    ranking.value = null
    return
  }
  loading.value = true
  try {
    const { data } = await client.get(`/exams/${selectedExamID.value}/ranking`)
    ranking.value = data
    lastLoadedAt.value = new Date()
  } finally {
    loading.value = false
  }
}

function examOptionLabel(exam: any) {
  const prefix = [exam.course_code, exam.class_name].filter(Boolean).join(' / ')
  return `${prefix ? `${prefix} · ` : ''}${exam.title}`
}

function examStatus(exam: any) {
  if (!exam) return '-'
  const now = Date.now()
  if (exam.starts_at && new Date(exam.starts_at).getTime() > now) return '未开始'
  if (exam.ends_at && new Date(exam.ends_at).getTime() <= now) return '已结束'
  return '进行中'
}

function problemCell(row: any, problemID: number) {
  return row.problems?.find((item: any) => item.problem_id === problemID)
}

function scoreText(cell: any) {
  if (!cell) return '-'
  if (cell.pending) return '待评分'
  if (!cell.score_ready) return '-'
  return `${cell.best_score} / ${cell.max_score}`
}

function statusText(cell: any) {
  if (!cell?.status) return '未提交'
  if (cell.pending) return '评分中'
  return String(cell.status).replace(/_/g, ' ')
}

function resetTimer() {
  if (refreshTimer) window.clearInterval(refreshTimer)
  refreshTimer = undefined
  if (!autoRefresh.value) return
  refreshTimer = window.setInterval(loadRanking, 5000)
}

watch(autoRefresh, resetTimer)
watch(selectedExamID, loadRanking)

onMounted(async () => {
  await loadExams()
  await loadRanking()
  resetTimer()
})

onBeforeUnmount(() => {
  if (refreshTimer) window.clearInterval(refreshTimer)
})
</script>

<style scoped>
.exam-ranking-page {
  min-width: 0;
}

.exam-select {
  width: min(420px, 48vw);
}

.scoreboard-hero {
  display: flex;
  align-items: stretch;
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 16px;
  background: linear-gradient(135deg, #101828, #1d2939 58%, #344054);
  color: #f8fafc;
}

.scoreboard-hero h3 {
  margin: 6px 0 8px;
  font-size: 28px;
  letter-spacing: 0;
}

.scoreboard-hero .muted,
.scoreboard-hero .eyebrow {
  color: #d0d5dd;
}

.ranking-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(72px, 1fr));
  gap: 10px;
  min-width: min(420px, 48vw);
}

.ranking-stats div {
  display: grid;
  gap: 4px;
  align-content: center;
  padding: 12px;
  border: 1px solid rgba(255, 255, 255, 0.18);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.08);
}

.ranking-stats strong {
  font-size: 24px;
}

.ranking-stats span,
.student-cell span,
.problem-score-cell small {
  color: var(--muted);
}

.ranking-panel {
  overflow: hidden;
}

.student-cell,
.problem-score-cell,
.empty-state {
  display: grid;
  gap: 4px;
}

.problem-score-cell strong {
  color: var(--text);
}

@media (max-width: 900px) {
  .scoreboard-hero {
    display: grid;
  }

  .ranking-stats {
    min-width: 0;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .exam-select {
    width: 100%;
  }
}
</style>
