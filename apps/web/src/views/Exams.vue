<template>
  <section class="page">
    <div class="page-header">
      <h2>考试</h2>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="router.push('/exams/new')">新建考试</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>

    <div class="panel">
      <el-table :data="items">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="course_id" label="课程" width="100" />
        <el-table-column prop="class_id" label="班级" width="100" />
        <el-table-column prop="title" label="标题" min-width="180" />
        <el-table-column prop="starts_at" label="开始" min-width="170" />
        <el-table-column prop="ends_at" label="结束" min-width="170" />
        <el-table-column v-if="canManage" label="模式" width="130">
          <template #default="{ row }">
            <el-tag v-if="row.settings?.manual_review" type="warning" effect="light">人工阅卷</el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="!canManage" label="状态" width="110">
          <template #default="{ row }"><el-tag :type="examStatusType(row)">{{ examStatusLabel(row) }}</el-tag></template>
        </el-table-column>
        <el-table-column v-if="!canManage" label="分数" width="120">
          <template #default="{ row }">{{ scoreText(row) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="260">
          <template #default="{ row }">
            <el-button size="small" type="primary" :disabled="!canManage && Boolean(row.finished_at)" @click="openDetail(row)">
              {{ !canManage && row.finished_at ? '已结束' : '进入' }}
            </el-button>
            <el-button v-if="canManage" size="small" @click="openReport(row)">完成情况</el-button>
            <el-button v-if="canManage" size="small" type="danger" plain @click="removeExam(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-drawer v-model="reportVisible" title="考试完成情况" size="86%">
      <div v-if="report" class="panel">
        <div class="toolbar report-toolbar">
          <el-tag v-if="report.manual_review" type="warning">人工阅卷</el-tag>
          <span class="muted">展开学生行可查看每题提交与评分</span>
          <el-button size="small" type="primary" :disabled="!reportEnded" :loading="exporting" @click="exportReport">导出 Excel</el-button>
        </div>
        <el-table :data="report.rows">
          <el-table-column type="expand">
            <template #default="{ row }">
              <el-table :data="row.problem_scores" size="small">
                <el-table-column label="题号" width="80">
                  <template #default="{ row: item }">{{ item.label || '-' }}</template>
                </el-table-column>
                <el-table-column prop="problem.title" label="题目" />
                <el-table-column prop="score" label="分值" width="80" />
                <el-table-column label="得分" width="120">
                  <template #default="{ row: item }">{{ item.score_ready ? item.best_score : '-' }}</template>
                </el-table-column>
                <el-table-column label="状态" width="130">
                  <template #default="{ row: item }">
                    <StatusBadge v-if="item.submission_id" :status="item.submission_status" />
                    <span v-else>-</span>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="190">
                  <template #default="{ row: item }">
                    <el-button v-if="report.manual_review && item.submission_id" size="small" @click="openGradeDialog(item)">阅卷</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </template>
          </el-table-column>
          <el-table-column prop="user.name" label="学生" />
          <el-table-column prop="user.student_no" label="学号" width="130" />
          <el-table-column label="状态" width="110">
            <template #default="{ row }"><el-tag :type="workStatusType(row.work_status)">{{ workStatusLabel(row.work_status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="总分" width="140">
            <template #default="{ row }">
              {{ row.score_ready ? `${row.total_score} / ${row.max_score}` : (row.work_status === 'submitted' ? '待评分' : '-') }}
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-drawer>

    <el-dialog v-model="gradeVisible" title="人工阅卷" width="980px">
      <div v-if="gradeSubmission" class="grade-grid">
        <div class="panel">
          <div class="section-title"><h3>学生代码</h3></div>
          <pre class="source-view">{{ gradeSubmission.submission.source_code }}</pre>
        </div>
        <div class="panel">
          <div class="section-title"><h3>参考判题</h3></div>
          <p><StatusBadge :status="gradeSubmission.submission.status" /> 参考分：{{ gradeSubmission.submission.score }}</p>
          <el-table :data="gradeSubmission.results || []" size="small" max-height="260">
            <el-table-column prop="case_name" label="测试点" />
            <el-table-column label="状态"><template #default="{ row }"><StatusBadge :status="row.status" /></template></el-table-column>
            <el-table-column prop="message" label="信息" />
          </el-table>
          <div class="grade-actions">
            <el-button :loading="grading" @click="runReferenceJudge">运行系统判题</el-button>
            <el-input-number v-model="manualScore" :min="0" :max="gradeMaxScore" />
            <span class="muted">/ {{ gradeMaxScore }}</span>
            <el-button type="primary" :loading="grading" @click="saveManualGrade">保存分数</el-button>
          </div>
        </div>
      </div>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { client } from '../api/client'
import StatusBadge from '../components/StatusBadge.vue'
import { useAuthStore } from '../stores/auth'
import { useClassroomStore } from '../stores/classroom'

const auth = useAuthStore()
const classroom = useClassroomStore()
const router = useRouter()
const canManage = computed(() => auth.role !== 'student')
const items = ref<any[]>([])
const reportVisible = ref(false)
const gradeVisible = ref(false)
const grading = ref(false)
const report = ref<any>(null)
const gradeSubmission = ref<any>(null)
const gradeProblemScore = ref<any>(null)
const manualScore = ref(0)
const gradeMaxScore = computed(() => gradeProblemScore.value?.score || 100)
const exporting = ref(false)
const reportEnded = computed(() => {
  if (!report.value?.exam?.ends_at) return false
  return new Date(report.value.exam.ends_at).getTime() <= Date.now()
})

async function load() {
  const params = classroom.activeClassId ? { class_id: classroom.activeClassId } : {}
  const examsRes = await client.get('/exams', { params })
  items.value = examsRes.data
}

function openDetail(row: any) {
  if (!canManage.value && row.finished_at) {
    ElMessage.warning('考试已结束，不能再次进入')
    return
  }
  router.push(`/exams/${row.id}`)
}

async function openReport(row: any) {
  report.value = (await client.get(`/exams/${row.id}/report`)).data
  reportVisible.value = true
}

async function exportReport() {
  if (!report.value?.exam?.id) return
  exporting.value = true
  try {
    const { data } = await client.get(`/exams/${report.value.exam.id}/report/export`, { responseType: 'blob' })
    downloadBlob(data, `exam-${report.value.exam.id}-report.xlsx`)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    exporting.value = false
  }
}

async function removeExam(row: any) {
  try {
    await ElMessageBox.confirm('删除后学生端将不再显示该考试，历史提交会保留。确认删除？', '删除考试', { type: 'warning' })
    await client.delete(`/exams/${row.id}`)
    ElMessage.success('考试已删除')
    await load()
  } catch (err: any) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.error || err.message)
  }
}

async function openGradeDialog(problemScore: any) {
  gradeProblemScore.value = problemScore
  const { data } = await client.get(`/submissions/${problemScore.submission_id}`)
  gradeSubmission.value = data
  manualScore.value = data.submission.manual_score ?? problemScore.best_score ?? 0
  gradeVisible.value = true
}

async function runReferenceJudge() {
  if (!report.value || !gradeSubmission.value) return
  grading.value = true
  try {
    await client.post(`/exams/${report.value.exam.id}/submissions/${gradeSubmission.value.submission.id}/judge`)
    ElMessage.success('已加入参考判题队列')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    grading.value = false
  }
}

async function saveManualGrade() {
  if (!report.value || !gradeSubmission.value) return
  grading.value = true
  try {
    await client.put(`/exams/${report.value.exam.id}/submissions/${gradeSubmission.value.submission.id}/grade`, { score: manualScore.value })
    ElMessage.success('分数已保存')
    gradeVisible.value = false
    await openReport(report.value.exam)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    grading.value = false
  }
}

function scoreText(row: any) {
  if (row.finished_at && row.work_status !== 'submitted') return `${row.total_score || 0} / ${row.max_score || 0}`
  if (row.work_status !== 'submitted') return '-'
  return row.score_ready ? `${row.total_score} / ${row.max_score}` : '待评分'
}

function examStatusLabel(row: any) {
  if (row.finished_at) return '已结束'
  return workStatusLabel(row.work_status)
}

function examStatusType(row: any): 'success' | 'warning' | 'info' {
  if (row.finished_at) return 'info'
  return workStatusType(row.work_status)
}

function workStatusLabel(status: string) {
  if (status === 'submitted') return '已提交'
  if (status === 'unsubmitted') return '未提交'
  return '未尝试'
}

function workStatusType(status: string): 'success' | 'warning' | 'info' {
  if (status === 'submitted') return 'success'
  if (status === 'unsubmitted') return 'warning'
  return 'info'
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

watch(() => classroom.activeClassId, load)

onMounted(async () => {
  await classroom.load()
  await load()
})
</script>

<style scoped>
.report-toolbar,
.grade-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.grade-grid {
  display: grid;
  grid-template-columns: minmax(320px, 1fr) minmax(320px, 1fr);
  gap: 14px;
}

.source-view {
  max-height: 520px;
  overflow: auto;
  padding: 12px;
  border-radius: 8px;
  background: #0f172a;
  color: #e2e8f0;
}

@media (max-width: 900px) {
  .grade-grid {
    grid-template-columns: 1fr;
  }
}
</style>
