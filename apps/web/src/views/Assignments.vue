<template>
  <section class="page">
    <div class="page-header">
      <h2>作业</h2>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="openDialog">新建作业</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>

    <div class="assignment-summary">
      <div v-for="item in assignmentStats" :key="item.label" class="panel assignment-stat">
        <strong>{{ item.value }}</strong>
        <span class="muted">{{ item.label }}</span>
      </div>
    </div>

    <div class="panel assignment-filters">
      <el-input v-model="filters.keyword" clearable placeholder="搜索标题、描述、课程或班级" />
      <el-select v-if="!canManage" v-model="filters.status" placeholder="状态">
        <el-option
          v-for="option in assignmentStatusOptions"
          :key="option.value"
          :label="option.label"
          :value="option.value"
        />
      </el-select>
      <el-button @click="resetFilters">重置</el-button>
      <span class="muted filter-count">{{ filteredItems.length }} / {{ items.length }}</span>
    </div>

    <div class="panel">
      <el-table :data="filteredItems">
        <el-table-column label="课程" min-width="150">
          <template #default="{ row }">{{ courseText(row) }}</template>
        </el-table-column>
        <el-table-column label="班级" min-width="120">
          <template #default="{ row }">{{ row.class_name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="title" label="标题" min-width="180" />
        <el-table-column label="截止时间" min-width="210">
          <template #default="{ row }">
            <div class="due-cell">
              <span>{{ formatDateTime(row.due_at) }}</span>
              <el-tag size="small" :type="assignmentState(row).type">
                {{ assignmentState(row).label }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="题量" width="90">
          <template #default="{ row }">{{ assignmentProblemCount(row) }}</template>
        </el-table-column>
        <el-table-column v-if="!canManage" label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="workStatusType(row.work_status)">{{ workStatusLabel(row.work_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="!canManage" label="分数" width="120">
          <template #default="{ row }">{{ scoreText(row) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="260">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="openDetail(row)">进入</el-button>
            <el-button v-if="canManage" size="small" @click="openReport(row)">完成情况</el-button>
            <el-button v-if="canManage" size="small" type="danger" plain @click="removeAssignment(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialogVisible" title="新建作业" width="860px">
      <el-form :model="form" label-width="90px">
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
        <el-form-item label="标题"><el-input v-model="form.title" placeholder="第一次作业" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="开始时间"><el-date-picker v-model="form.starts_at" type="datetime" format="YYYY-MM-DD HH:mm:ss" style="width: 100%" /></el-form-item>
        <el-form-item label="截止时间"><el-date-picker v-model="form.due_at" type="datetime" format="YYYY-MM-DD HH:mm:ss" style="width: 100%" /></el-form-item>
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
          <el-table-column label="分值" width="160">
            <template #default="{ row }"><el-input-number v-model="row.score" :min="1" :max="1000" /></template>
          </el-table-column>
          <el-table-column label="操作" width="90">
            <template #default="{ $index }"><el-button size="small" text type="danger" @click="selectedProblems.splice($index, 1)">移除</el-button></template>
          </el-table-column>
        </el-table>
        <div class="total-line">总分：{{ selectedTotalScore }}</div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="reportVisible" title="作业完成情况" size="82%">
      <div v-if="report" class="panel">
        <el-table :data="report.rows">
          <el-table-column type="expand">
            <template #default="{ row }">
              <el-table :data="row.problem_scores" size="small">
                <el-table-column prop="problem.title" label="题目" />
                <el-table-column prop="score" label="分值" width="80" />
                <el-table-column label="得分" width="120">
                  <template #default="{ row: item }">{{ item.score_ready ? item.best_score : '-' }}</template>
                </el-table-column>
                <el-table-column label="最近提交" width="170">
                  <template #default="{ row: item }">{{ formatDateTime(item.submitted_at) }}</template>
                </el-table-column>
              </el-table>
            </template>
          </el-table-column>
          <el-table-column prop="user.name" label="学生" />
          <el-table-column prop="user.student_no" label="学号" width="130" />
          <el-table-column label="状态" width="110">
            <template #default="{ row }"><el-tag :type="workStatusType(row.work_status)">{{ workStatusLabel(row.work_status) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="总分" width="120">
            <template #default="{ row }">{{ row.score_ready ? `${row.total_score} / ${row.max_score}` : '-' }}</template>
          </el-table-column>
        </el-table>
      </div>
    </el-drawer>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { client, type PreparedProblem, type Problem } from '../api/client'
import {
  assignmentMatchesFilters,
  assignmentProblemCount,
  assignmentState,
  assignmentStatusOptions,
  formatDateTime,
  scoreText,
  workStatusLabel,
  workStatusType,
  type AssignmentFilters
} from '../features/assignments/assignmentMeta'
import { problemDisplayCode } from '../features/problems/problemMeta'
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
const filters = reactive<AssignmentFilters>({
  keyword: '',
  status: 'all'
})
const dialogVisible = ref(false)
const reportVisible = ref(false)
const saving = ref(false)
const problemSource = ref<'class' | 'prepared'>('class')
const problemPickID = ref<number>()
const selectedProblems = ref<SelectedProblem[]>([])
const report = ref<any>(null)

const form = reactive<any>({
  course_id: undefined,
  class_id: undefined,
  title: '',
  description: '',
  starts_at: null,
  due_at: null
})

const problemOptions = computed(() => {
  if (problemSource.value === 'prepared') {
    return preparedProblems.value.map((item) => {
      const code = item.problem ? problemDisplayCode(item.problem) : '未编号'
      return { value: item.problem_id, label: `[预备] ${code}. ${item.problem?.title || '未知题目'}`, title: item.problem?.title, source: '预备' }
    })
  }
  return problems.value.map((problem) => ({ value: problem.id, label: `[题库] ${problemDisplayCode(problem)}. ${problem.title}`, title: problem.title, source: '题库' }))
})
const selectedTotalScore = computed(() => selectedProblems.value.reduce((sum, item) => sum + Number(item.score || 0), 0))
const filteredItems = computed(() => items.value.filter((item) => assignmentMatchesFilters(item, filters)))
const assignmentStats = computed(() => {
  const active = items.value.filter((item) => assignmentState(item).label === '进行中').length
  const problemCount = items.value.reduce((sum, item) => sum + assignmentProblemCount(item), 0)
  if (canManage.value) {
    return [
      { label: '作业总数', value: items.value.length },
      { label: '进行中', value: active },
      { label: '题目总量', value: problemCount }
    ]
  }
  return [
    { label: '作业总数', value: items.value.length },
    { label: '已提交', value: items.value.filter((item) => item.work_status === 'submitted').length },
    { label: '进行中', value: active }
  ]
})

async function load() {
  const params = classroom.activeClassId ? { class_id: classroom.activeClassId } : {}
  const assignmentsRes = await client.get('/assignments', { params })
  items.value = assignmentsRes.data
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

function resetFilters() {
  filters.keyword = ''
  filters.status = 'all'
}

function courseText(row: any) {
  return [row.course_code, row.course_name].filter(Boolean).join(' ') || '-'
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
  if (selectedProblems.value.some((item) => item.source === '预备') && !form.due_at) {
    ElMessage.error('使用预备题创建作业时必须填写截止时间')
    return
  }
  saving.value = true
  try {
    await client.post('/assignments', {
      course_id: form.course_id,
      class_id: form.class_id,
      title: form.title,
      description: form.description,
      starts_at: form.starts_at,
      due_at: form.due_at,
      problems: selectedProblems.value.map((item) => ({ problem_id: item.problem_id, score: item.score }))
    })
    ElMessage.success('作业已创建')
    dialogVisible.value = false
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    saving.value = false
  }
}

function openDetail(row: any) {
  router.push(`/assignments/${row.id}`)
}

async function openReport(row: any) {
  report.value = (await client.get(`/assignments/${row.id}/report`)).data
  reportVisible.value = true
}

async function removeAssignment(row: any) {
  try {
    await ElMessageBox.confirm('删除后学生端将不再显示该作业，历史提交会保留。确认删除？', '删除作业', { type: 'warning' })
    await client.delete(`/assignments/${row.id}`)
    ElMessage.success('作业已删除')
    await load()
  } catch (err: any) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.error || err.message)
  }
}

function reset() {
  form.course_id = undefined
  form.class_id = undefined
  form.title = ''
  form.description = ''
  form.starts_at = null
  form.due_at = null
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
.assignment-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(120px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.assignment-stat {
  display: grid;
  gap: 6px;
}

.assignment-stat strong {
  color: var(--text);
  font-size: 26px;
}

.assignment-filters {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) 140px auto auto;
  gap: 10px;
  align-items: center;
  margin-bottom: 16px;
}

.filter-count {
  justify-self: end;
  white-space: nowrap;
}

.due-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

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
  margin-left: 90px;
  width: calc(100% - 90px);
}

.total-line {
  margin: 10px 0 0 90px;
  font-weight: 700;
}

@media (max-width: 760px) {
  .assignment-summary,
  .assignment-filters,
  .problem-add {
    grid-template-columns: 1fr;
  }

  .filter-count {
    justify-self: start;
  }

  .selected-problems,
  .total-line {
    width: 100%;
    margin-left: 0;
  }
}
</style>
