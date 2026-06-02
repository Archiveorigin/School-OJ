<template>
  <section class="page">
    <div class="page-header">
      <h2>作业</h2>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="openDialog">新建作业</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>

    <div class="panel">
      <el-table :data="items">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="course_id" label="课程" width="100" />
        <el-table-column prop="class_id" label="班级" width="100" />
        <el-table-column prop="title" label="标题" min-width="180" />
        <el-table-column prop="due_at" label="截止时间" min-width="180" />
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
            <el-button v-if="canManage" size="small" type="danger" plain @click="removeAssignment(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dialogVisible" title="新建作业" width="860px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="课程">
          <el-select v-model="form.course_id" style="width: 100%" disabled>
            <el-option
              v-for="course in courses"
              :key="course.id"
              :label="`${course.code} ${course.name}`"
              :value="course.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="班级">
          <el-select v-model="form.class_id" style="width: 100%" @change="syncCourseFromClass">
            <el-option
              v-for="item in classroom.classes"
              :key="item.class_id"
              :label="`${item.course_code} / ${item.class_name}`"
              :value="item.class_id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="标题"><el-input v-model="form.title" placeholder="第一次作业" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="开始时间"><el-date-picker v-model="form.starts_at" type="datetime" style="width: 100%" /></el-form-item>
        <el-form-item label="截止时间"><el-date-picker v-model="form.due_at" type="datetime" style="width: 100%" /></el-form-item>
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
            <template #default="{ row }">
              <el-input-number v-model="row.score" :min="1" :max="1000" />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="90">
            <template #default="{ $index }">
              <el-button size="small" text type="danger" @click="selectedProblems.splice($index, 1)">移除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="total-line">总分：{{ selectedTotalScore }}</div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" :title="detailTitle" size="92%">
      <div v-if="detail" class="workbench">
        <div class="workbench-head">
          <div>
            <h3>{{ detail.assignment.title }}</h3>
            <span class="muted">截止时间：{{ detail.assignment.due_at || '-' }}</span>
          </div>
          <div class="toolbar">
            <el-tag v-if="detail.not_started" type="warning">未开始</el-tag>
            <el-tag v-else-if="detail.closed" type="info">已截止</el-tag>
            <el-tag v-else type="success">可提交</el-tag>
            <el-tag>{{ workStatusLabel(detail.work_status) }}</el-tag>
            <strong>{{ detail.score_ready ? `${detail.total_score} / ${detail.max_score}` : '分数计算中' }}</strong>
          </div>
        </div>

        <div class="coding-grid">
          <aside class="problem-rail">
            <button
              v-for="entry in detail.problems"
              :key="entry.problem.id"
              type="button"
              class="problem-pick"
              :class="{ active: activeProblem?.id === entry.problem.id }"
              @click="selectDetailProblem(entry)"
            >
              <strong>{{ entry.problem.title }}</strong>
              <span>{{ entry.score }} 分 · {{ problemScoreText(entry.problem.id) }}</span>
            </button>
          </aside>

          <main v-if="activeProblem" class="statement-panel">
            <div class="panel-title">
              <h3>{{ activeProblem.title }}</h3>
              <span>{{ activeEntry?.score }} 分</span>
            </div>
            <p class="muted">{{ activeProblem.time_limit_ms }} ms / {{ activeProblem.memory_limit_mb }} MB / {{ activeProblem.output_limit_kb }} KB</p>
            <p class="statement">{{ activeProblem.statement }}</p>
          </main>

          <section v-if="activeProblem" class="editor-panel">
            <div class="toolbar editor-toolbar">
              <el-select v-model="language" style="width: 130px">
                <el-option label="C++17" value="cpp" />
                <el-option label="C" value="c" />
                <el-option label="Python" value="python" />
                <el-option label="Java" value="java" />
              </el-select>
              <el-button type="primary" :loading="submitting" :disabled="!detail.can_submit" @click="submitSolution">提交</el-button>
            </div>
            <CodeEditor v-model="source" :language="language" />
            <div v-if="live" class="live"><StatusBadge :status="live.status" /> 分数 {{ live.score }}，{{ live.message }}</div>
          </section>
        </div>

        <div class="panel history-panel">
          <div class="section-title"><h3>当前题提交记录</h3></div>
          <el-table :data="history" size="small">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="language" label="语言" width="90" />
            <el-table-column label="状态" width="130"><template #default="{ row }"><StatusBadge :status="row.status" /></template></el-table-column>
            <el-table-column prop="score" label="原始分" width="90" />
            <el-table-column label="时间" min-width="160"><template #default="{ row }">{{ row.created_at }}</template></el-table-column>
          </el-table>
        </div>
      </div>
    </el-drawer>

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
                <el-table-column label="提交" width="120">
                  <template #default="{ row: item }">{{ item.submission_id ? `#${item.submission_id}` : '-' }}</template>
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
import { client, sseUrl, type PreparedProblem, type Problem, type Submission } from '../api/client'
import CodeEditor from '../components/CodeEditor.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useAuthStore } from '../stores/auth'
import { useClassroomStore } from '../stores/classroom'

type DetailProblem = { problem: Problem; score: number; problem_id: number }
type SelectedProblem = { problem_id: number; title: string; source: string; score: number }

const auth = useAuthStore()
const classroom = useClassroomStore()
const canManage = computed(() => auth.role !== 'student')
const items = ref<any[]>([])
const courses = ref<any[]>([])
const problems = ref<Problem[]>([])
const preparedProblems = ref<PreparedProblem[]>([])
const dialogVisible = ref(false)
const detailVisible = ref(false)
const reportVisible = ref(false)
const saving = ref(false)
const submitting = ref(false)
const problemSource = ref<'class' | 'prepared'>('class')
const problemPickID = ref<number>()
const selectedProblems = ref<SelectedProblem[]>([])
const detail = ref<any>(null)
const report = ref<any>(null)
const activeEntry = ref<DetailProblem | null>(null)
const activeProblem = computed(() => activeEntry.value?.problem || null)
const language = ref('cpp')
const live = ref<any>(null)
const history = ref<Submission[]>([])
const source = ref('')

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
    return preparedProblems.value.map((item) => ({ value: item.problem_id, label: `[预备] ${item.problem_id}. ${item.problem?.title}`, title: item.problem?.title, source: '预备' }))
  }
  return problems.value.map((problem) => ({ value: problem.id, label: `[题库] ${problem.id}. ${problem.title}`, title: problem.title, source: '题库' }))
})
const selectedTotalScore = computed(() => selectedProblems.value.reduce((sum, item) => sum + Number(item.score || 0), 0))
const detailTitle = computed(() => detail.value?.assignment?.title || '作业')

async function load() {
  const params = classroom.activeClassId ? { class_id: classroom.activeClassId } : {}
  const requests: Promise<any>[] = [client.get('/assignments', { params }), client.get('/courses'), client.get('/problems', { params })]
  if (canManage.value) requests.push(client.get('/prepared-problems'))
  const [assignmentsRes, coursesRes, problemsRes, preparedRes] = await Promise.all(requests)
  items.value = assignmentsRes.data
  courses.value = coursesRes.data
  problems.value = problemsRes.data
  preparedProblems.value = preparedRes?.data || []
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

async function openDetail(row: any) {
  try {
    detail.value = (await client.get(`/assignments/${row.id}`)).data
    activeEntry.value = detail.value.problems?.[0] || null
    live.value = null
    detailVisible.value = true
    loadDraft()
    await loadHistory()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
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

function selectDetailProblem(entry: DetailProblem) {
  activeEntry.value = entry
  live.value = null
  loadDraft()
  loadHistory()
}

async function submitSolution() {
  if (!activeProblem.value || !detail.value) return
  submitting.value = true
  try {
    const { data } = await client.post('/submissions', {
      problem_id: activeProblem.value.id,
      assignment_id: detail.value.assignment.id,
      language: language.value,
      source_code: source.value
    })
    watchSubmission(data.id)
    await loadHistory()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    submitting.value = false
  }
}

function watchSubmission(id: number) {
  const es = new EventSource(sseUrl(`/submissions/${id}/events`))
  es.addEventListener('status', async (event) => {
    live.value = JSON.parse((event as MessageEvent).data)
    if (!['queued', 'running'].includes(live.value.status)) {
      es.close()
      await refreshDetail()
      await loadHistory()
    }
  })
}

async function refreshDetail() {
  if (!detail.value) return
  const activeID = activeProblem.value?.id
  detail.value = (await client.get(`/assignments/${detail.value.assignment.id}`)).data
  activeEntry.value = detail.value.problems.find((entry: DetailProblem) => entry.problem.id === activeID) || detail.value.problems[0] || null
}

async function loadHistory() {
  if (!detail.value || !activeProblem.value) return
  history.value = (await client.get('/submissions', { params: { problem_id: activeProblem.value.id, assignment_id: detail.value.assignment.id } })).data
}

function loadDraft() {
  if (!detail.value || !activeProblem.value) return
  source.value = localStorage.getItem(draftKey()) || defaultSource(language.value)
}

function saveDraft() {
  if (!detail.value || !activeProblem.value) return
  localStorage.setItem(draftKey(), source.value)
}

function draftKey() {
  return `school-oj-draft:assignment:${detail.value.assignment.id}:${activeProblem.value?.id}:${language.value}`
}

function defaultSource(lang: string) {
  if (lang === 'python') return 'a, b = map(int, input().split())\nprint(a + b)\n'
  if (lang === 'java') return 'import java.util.*;\npublic class Main { public static void main(String[] args) { Scanner sc = new Scanner(System.in); long a = sc.nextLong(), b = sc.nextLong(); System.out.println(a + b); } }\n'
  if (lang === 'c') return '#include <stdio.h>\nint main(){ long long a,b; scanf("%lld%lld",&a,&b); printf("%lld\\n", a+b); return 0; }\n'
  return '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ long long a,b; cin>>a>>b; cout<<a+b<<"\\n"; return 0; }\n'
}

function scoreForProblem(problemID: number) {
  return detail.value?.problem_scores?.find((item: any) => item.problem.id === problemID)
}

function problemScoreText(problemID: number) {
  const item = scoreForProblem(problemID)
  if (!item?.submission_id) return '未提交'
  if (!item.score_ready) return '计算中'
  return `${item.best_score} / ${item.score}`
}

function scoreText(row: any) {
  if (row.work_status !== 'submitted') return '-'
  return row.score_ready ? `${row.total_score} / ${row.max_score}` : '计算中'
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
  form.due_at = null
  selectedProblems.value = []
  problemPickID.value = undefined
}

watch(() => classroom.activeClassId, load)
watch(language, loadDraft)
watch(source, saveDraft)

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
  margin-left: 90px;
  width: calc(100% - 90px);
}

.total-line {
  margin: 10px 0 0 90px;
  font-weight: 700;
}

.workbench {
  display: grid;
  gap: 14px;
}

.workbench-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.workbench-head h3 {
  margin: 0 0 4px;
}

.coding-grid {
  display: grid;
  grid-template-columns: 230px minmax(260px, 0.9fr) minmax(360px, 1.1fr);
  gap: 14px;
  min-height: 560px;
}

.problem-rail,
.statement-panel,
.editor-panel,
.history-panel {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  padding: 12px;
}

.problem-rail {
  display: grid;
  align-content: start;
  gap: 8px;
}

.problem-pick {
  display: grid;
  gap: 4px;
  width: 100%;
  padding: 10px;
  border: 1px solid #d9dee8;
  border-radius: 8px;
  background: transparent;
  color: var(--text);
  text-align: left;
  cursor: pointer;
}

.problem-pick.active {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(10, 94, 166, 0.12);
}

.problem-pick span {
  color: #6b7280;
  font-size: 12px;
}

.panel-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.panel-title h3 {
  margin: 0;
}

.statement {
  white-space: pre-wrap;
  line-height: 1.7;
}

.editor-panel {
  display: grid;
  grid-template-rows: auto minmax(420px, 1fr) auto;
  gap: 10px;
}

.editor-toolbar {
  justify-content: flex-end;
}

.live {
  display: flex;
  gap: 10px;
  align-items: center;
}

@media (max-width: 1100px) {
  .coding-grid {
    grid-template-columns: 1fr;
  }
}
</style>
