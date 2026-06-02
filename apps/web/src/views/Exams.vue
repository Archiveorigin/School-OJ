<template>
  <section class="page">
    <div class="page-header">
      <h2>考试</h2>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="openDialog">新建考试</el-button>
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

    <el-dialog v-model="dialogVisible" title="新建考试" width="880px">
      <el-form :model="form" label-width="96px">
        <el-form-item label="课程">
          <el-select v-model="form.course_id" style="width: 100%" disabled>
            <el-option v-for="course in courses" :key="course.id" :label="`${course.code} ${course.name}`" :value="course.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="班级">
          <el-select v-model="form.class_id" style="width: 100%" @change="syncCourseFromClass">
            <el-option v-for="item in classroom.classes" :key="item.class_id" :label="`${item.course_code} / ${item.class_name}`" :value="item.class_id" />
          </el-select>
        </el-form-item>
        <el-form-item label="标题"><el-input v-model="form.title" placeholder="期中考试" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="开始时间"><el-date-picker v-model="form.starts_at" type="datetime" style="width: 100%" /></el-form-item>
        <el-form-item label="结束时间"><el-date-picker v-model="form.ends_at" type="datetime" style="width: 100%" /></el-form-item>
        <el-form-item label="阅卷方式">
          <el-checkbox v-model="form.manual_review">提交后判题，教师人工确认分数</el-checkbox>
        </el-form-item>
        <el-form-item label="考试退出">
          <span class="muted">学生进入考试后必须点击“结束考试”才能退出，结束后不能再次进入。</span>
        </el-form-item>
        <el-form-item label="添加题目">
          <div class="problem-add">
            <el-radio-group v-model="problemSource" @change="problemPickID = undefined">
              <el-radio-button label="class">班级题库</el-radio-button>
              <el-radio-button label="prepared">预备题库</el-radio-button>
            </el-radio-group>
            <el-select v-model="problemPickID" filterable placeholder="选择题目" class="problem-select">
              <el-option v-for="option in problemOptions" :key="option.value" :label="option.label" :value="option.value" />
            </el-select>
            <el-button @click="addSelectedProblem">添加</el-button>
          </div>
        </el-form-item>
        <el-table :data="selectedProblems" size="small" class="selected-problems">
          <el-table-column prop="source" label="来源" width="90" />
          <el-table-column prop="title" label="题目" />
          <el-table-column label="分值" width="160"><template #default="{ row }"><el-input-number v-model="row.score" :min="1" :max="1000" /></template></el-table-column>
          <el-table-column label="操作" width="90"><template #default="{ $index }"><el-button size="small" text type="danger" @click="selectedProblems.splice($index, 1)">移除</el-button></template></el-table-column>
        </el-table>
        <div class="total-line">总分：{{ selectedTotalScore }}</div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="reportVisible" title="考试完成情况" size="86%">
      <div v-if="report" class="panel">
        <div class="toolbar report-toolbar">
          <el-tag v-if="report.manual_review" type="warning">人工阅卷</el-tag>
          <span class="muted">展开学生行可查看每题提交与评分</span>
        </div>
        <el-table :data="report.rows">
          <el-table-column type="expand">
            <template #default="{ row }">
              <el-table :data="row.problem_scores" size="small">
                <el-table-column prop="problem.title" label="题目" />
                <el-table-column prop="score" label="分值" width="80" />
                <el-table-column label="得分" width="120"><template #default="{ row: item }">{{ item.score_ready ? item.best_score : '-' }}</template></el-table-column>
                <el-table-column label="状态" width="130"><template #default="{ row: item }"><StatusBadge v-if="item.submission_id" :status="item.submission_status" /><span v-else>-</span></template></el-table-column>
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
          <el-table-column label="状态" width="110"><template #default="{ row }"><el-tag :type="workStatusType(row.work_status)">{{ workStatusLabel(row.work_status) }}</el-tag></template></el-table-column>
          <el-table-column label="总分" width="140"><template #default="{ row }">{{ row.score_ready ? `${row.total_score} / ${row.max_score}` : (row.work_status === 'submitted' ? '待评分' : '-') }}</template></el-table-column>
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
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { client, type PreparedProblem, type Problem } from '../api/client'
import StatusBadge from '../components/StatusBadge.vue'
import { useAuthStore } from '../stores/auth'
import { useClassroomStore } from '../stores/classroom'

type SelectedProblem = { problem_id: number; title: string; source: string; score: number }

const auth = useAuthStore()
const classroom = useClassroomStore()
const router = useRouter()
const canManage = computed(() => auth.role !== 'student')
const items = ref<any[]>([])
const courses = ref<any[]>([])
const problems = ref<Problem[]>([])
const preparedProblems = ref<PreparedProblem[]>([])
const dialogVisible = ref(false)
const reportVisible = ref(false)
const gradeVisible = ref(false)
const saving = ref(false)
const grading = ref(false)
const problemSource = ref<'class' | 'prepared'>('class')
const problemPickID = ref<number>()
const selectedProblems = ref<SelectedProblem[]>([])
const report = ref<any>(null)
const gradeSubmission = ref<any>(null)
const gradeProblemScore = ref<any>(null)
const manualScore = ref(0)
const gradeMaxScore = computed(() => gradeProblemScore.value?.score || 100)

const form = reactive<any>({
  course_id: undefined,
  class_id: undefined,
  title: '',
  description: '',
  starts_at: null,
  ends_at: null,
  manual_review: false
})

const problemOptions = computed(() => {
  if (problemSource.value === 'prepared') {
    return preparedProblems.value.map((item) => ({ value: item.problem_id, label: `[预备] ${item.problem_id}. ${item.problem?.title}`, title: item.problem?.title, source: '预备' }))
  }
  return problems.value.map((problem) => ({ value: problem.id, label: `[题库] ${problem.id}. ${problem.title}`, title: problem.title, source: '题库' }))
})
const selectedTotalScore = computed(() => selectedProblems.value.reduce((sum, item) => sum + Number(item.score || 0), 0))

async function load() {
  const params = classroom.activeClassId ? { class_id: classroom.activeClassId } : {}
  const examsRes = await client.get('/exams', { params })
  items.value = examsRes.data
  if (canManage.value) {
    const [coursesRes, problemsRes, preparedRes] = await Promise.all([
      client.get('/courses'),
      client.get('/problems', { params }),
      client.get('/prepared-problems')
    ])
    courses.value = coursesRes.data
    problems.value = problemsRes.data
    preparedProblems.value = preparedRes.data
  } else {
    courses.value = []
    problems.value = []
    preparedProblems.value = []
  }
}

function openDialog() {
  reset()
  form.class_id = classroom.activeClassId || classroom.classes[0]?.class_id
  syncCourseFromClass()
  dialogVisible.value = true
}

function syncCourseFromClass() {
  const item = classroom.classes.find((entry) => entry.class_id === form.class_id)
  form.course_id = item?.course_id
  selectedProblems.value = []
  loadClassProblems()
}

async function loadClassProblems() {
  if (!form.class_id) return
  problems.value = (await client.get('/problems', { params: { class_id: form.class_id } })).data
}

function addSelectedProblem() {
  const option = problemOptions.value.find((item) => item.value === problemPickID.value)
  if (!option || selectedProblems.value.some((item) => item.problem_id === option.value)) return
  selectedProblems.value.push({ problem_id: option.value, title: option.title || option.label, source: option.source, score: 100 })
  problemPickID.value = undefined
}

async function submitCreate() {
  if (!form.class_id || !form.course_id || !form.title || selectedProblems.value.length === 0) {
    ElMessage.error('请选择班级、填写标题并选择题目')
    return
  }
  if (selectedProblems.value.some((item) => item.source === '预备') && !form.ends_at) {
    ElMessage.error('使用预备题创建考试时必须填写结束时间')
    return
  }
  saving.value = true
  try {
    await client.post('/exams', {
      course_id: form.course_id,
      class_id: form.class_id,
      title: form.title,
      description: form.description,
      starts_at: form.starts_at,
      ends_at: form.ends_at,
      manual_review: form.manual_review,
      problems: selectedProblems.value.map((item) => ({ problem_id: item.problem_id, score: item.score }))
    })
    ElMessage.success('考试已创建')
    dialogVisible.value = false
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    saving.value = false
  }
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

function reset() {
  form.course_id = undefined
  form.class_id = undefined
  form.title = ''
  form.description = ''
  form.starts_at = null
  form.ends_at = null
  form.manual_review = false
  selectedProblems.value = []
  problemPickID.value = undefined
}

watch(() => classroom.activeClassId, load)

onMounted(async () => {
  await classroom.load()
  await load()
})
</script>

<style scoped>
.problem-add {
  display: grid;
  grid-template-columns: auto minmax(240px, 1fr) auto;
  gap: 10px;
  width: 100%;
}

.problem-select {
  width: 100%;
}

.selected-problems {
  margin-left: 96px;
  width: calc(100% - 96px);
}

.total-line {
  margin: 10px 0 0 96px;
  font-weight: 700;
}

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
