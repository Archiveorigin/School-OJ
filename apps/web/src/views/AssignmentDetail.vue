<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h2>{{ detail?.assignment?.title || '作业' }}</h2>
        <p v-if="detail" class="muted">截止时间：{{ formatDateTime(detail.assignment.due_at) }}</p>
      </div>
      <div class="toolbar">
        <el-tag v-if="detail?.not_started" type="warning">未开始</el-tag>
        <el-tag v-else-if="detail?.closed" type="info">已截止</el-tag>
        <el-tag v-else-if="detail" type="success">可提交</el-tag>
        <el-tag v-if="detail">{{ workStatusLabel(detail.work_status) }}</el-tag>
        <strong v-if="detail">{{ detail.score_ready ? `${detail.total_score} / ${detail.max_score}` : '分数计算中' }}</strong>
        <el-button @click="router.push(assignmentListPath)">返回列表</el-button>
      </div>
    </div>

    <div v-if="detail" class="workbench">
      <div class="problem-strip">
        <button
          v-for="entry in detail.problems"
          :key="entry.problem.id"
          type="button"
          class="problem-pick"
          :class="{ active: activeProblem?.id === entry.problem.id }"
          @click="selectDetailProblem(entry)"
        >
          <strong>{{ problemDisplayCode(entry.problem) }} · {{ entry.problem.title }}</strong>
          <span>{{ entry.score }} 分 · {{ problemScoreText(entry.problem.id) }}</span>
          <small v-if="entry.problem.deleted_at" class="muted">已下架</small>
        </button>
      </div>

      <ProblemStatementView
        v-if="activeProblem"
        :problem="activeProblem"
        :problem-number="problemDisplayCode(activeProblem)"
        :score="activeEntry?.score"
        :status-text="problemScoreText(activeProblem.id)"
        :status-type="problemStatusType(activeProblem.id)"
      />

      <section v-if="activeProblem && canManage" class="panel test-data-panel">
        <ProblemTestDownloads :problem-id="activeProblem.id" :problem-code="activeProblem.display_code" />
      </section>

      <section v-if="activeProblem" class="panel editor-panel">
        <SubmissionComposer
          v-model:language="language"
          v-model:source="source"
          :status="currentStatus"
          :message="currentMessage"
          :submitting="submitting"
          :disabled="!detail.can_submit"
          scope-text="本次代码仅计入当前作业"
          @submit="submitSolution"
        />
      </section>

      <div class="panel history-panel">
        <div class="section-title"><h3>全部提交记录</h3></div>
        <el-table :data="pagedHistory" size="small">
          <el-table-column label="题目" min-width="180">
            <template #default="{ row }">{{ problemTitle(row.problem_id) }}</template>
          </el-table-column>
          <el-table-column prop="language" label="语言" width="90" />
          <el-table-column label="状态" width="130"><template #default="{ row }"><StatusBadge :status="row.status" /></template></el-table-column>
          <el-table-column prop="score" label="原始分" width="90" />
          <el-table-column label="时间" min-width="170"><template #default="{ row }">{{ formatDateTime(row.created_at) }}</template></el-table-column>
        </el-table>
        <ListPagination v-model:page="historyPage" v-model:page-size="historyPageSize" :total="history.length" />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, openEventStream, type Problem, type Submission } from '../api/client'
import ListPagination from '../components/ListPagination.vue'
import ProblemStatementView from '../components/ProblemStatementView.vue'
import ProblemTestDownloads from '../components/ProblemTestDownloads.vue'
import StatusBadge from '../components/StatusBadge.vue'
import SubmissionComposer from '../components/SubmissionComposer.vue'
import { formatDateTime, workStatusLabel } from '../features/assignments/assignmentMeta'
import { problemDisplayCode } from '../features/problems/problemMeta'
import { useAuthStore } from '../stores/auth'

type DetailProblem = { problem: Problem; score: number; problem_id: number }
type EditorState = { language: string; source: string; live: any; dirty: boolean }

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
const detail = ref<any>(null)
const assignmentListPath = computed(() => {
  const courseID = detail.value?.assignment?.course_id
  return courseID ? `/my/courses/${courseID}/assignments` : '/my/courses'
})
const activeEntry = ref<DetailProblem | null>(null)
const activeProblem = computed(() => activeEntry.value?.problem || null)
const history = ref<Submission[]>([])
const historyPage = ref(1)
const historyPageSize = ref(10)
const submitting = ref(false)
const editorStates = reactive<Record<number, EditorState>>({})

const activeState = computed(() => {
  if (!activeProblem.value) return null
  return ensureEditorState(activeProblem.value.id)
})
const language = computed({
  get: () => activeState.value?.language || 'cpp',
  set: (value: string) => {
    if (!activeProblem.value || !activeState.value) return
    saveDraft(activeProblem.value.id, activeState.value.language, activeState.value.source)
    activeState.value.language = value
    activeState.value.source = preferredSubmission(activeProblem.value.id, value)?.source_code || ''
    activeState.value.dirty = false
  }
})
const source = computed({
  get: () => activeState.value?.source || '',
  set: (value: string) => {
    if (!activeProblem.value || !activeState.value) return
    activeState.value.source = value
    activeState.value.dirty = true
    saveDraft(activeProblem.value.id, activeState.value.language, value)
  }
})
const live = computed(() => activeState.value?.live)
const latestSubmission = computed(() => activeProblem.value ? preferredSubmission(activeProblem.value.id) : null)
const currentStatus = computed(() => live.value?.status || latestSubmission.value?.status || '')
const currentMessage = computed(() => live.value?.message || latestSubmission.value?.message || '')
const pagedHistory = computed(() => history.value.slice((historyPage.value - 1) * historyPageSize.value, historyPage.value * historyPageSize.value))

async function loadDetail() {
  const id = Number(route.params.id)
  if (!id) return
  try {
    detail.value = (await client.get(`/assignments/${id}`)).data
    activeEntry.value = detail.value.problems?.[0] || null
    if (activeProblem.value) ensureEditorState(activeProblem.value.id)
    await loadHistory()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
    router.push(assignmentListPath.value)
  }
}

function selectDetailProblem(entry: DetailProblem) {
  activeEntry.value = entry
  ensureEditorState(entry.problem.id).live = null
}

async function submitSolution() {
  if (!activeProblem.value || !detail.value || !activeState.value) return
  if (!activeState.value.source.trim()) {
    ElMessage.warning('请输入代码')
    return
  }
  const problemID = activeProblem.value.id
  submitting.value = true
  try {
    const { data } = await client.post('/submissions', {
      problem_id: problemID,
      assignment_id: detail.value.assignment.id,
      language: activeState.value.language,
      source_code: activeState.value.source
    })
    watchSubmission(data.id, problemID)
    await loadHistory()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    submitting.value = false
  }
}

function watchSubmission(id: number, problemID: number) {
  const es = openEventStream(`/submissions/${id}/events`)
  es.addEventListener('status', async (event) => {
    ensureEditorState(problemID).live = JSON.parse((event as MessageEvent).data)
    if (!['queued', 'running'].includes(ensureEditorState(problemID).live.status)) {
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
  if (!detail.value) return
  history.value = (await client.get('/submissions', { params: { assignment_id: detail.value.assignment.id } })).data
  hydrateEditorStatesFromHistory()
  clampHistoryPage()
}

function ensureEditorState(problemID: number) {
  if (!editorStates[problemID]) {
    editorStates[problemID] = { language: 'cpp', source: '', live: null, dirty: false }
  }
  return editorStates[problemID]
}

function saveDraft(problemID: number, lang: string, value: string) {
  if (!detail.value) return
  localStorage.setItem(draftKey(problemID, lang), value)
}

function draftKey(problemID: number, lang: string) {
  return `school-oj-draft:assignment:${detail.value.assignment.id}:${problemID}:${lang}`
}

function hydrateEditorStatesFromHistory() {
  for (const entry of detail.value?.problems || []) {
    const state = ensureEditorState(entry.problem.id)
    if (state.dirty) continue
    const latest = preferredSubmission(entry.problem.id)
    state.language = latest?.language || 'cpp'
    state.source = latest?.source_code || ''
  }
}

function preferredSubmission(problemID: number, selectedLanguage?: string) {
  return history.value.find((item) => item.user_id === auth.user?.id && item.problem_id === problemID && (!selectedLanguage || item.language === selectedLanguage)) || null
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

function problemStatusType(problemID: number): 'success' | 'warning' | 'info' | 'danger' {
  const item = scoreForProblem(problemID)
  if (!item?.submission_id) return 'info'
  if (!item.score_ready) return 'warning'
  if (item.best_score >= item.score) return 'success'
  if (item.best_score > 0) return 'warning'
  return 'danger'
}

function problemTitle(problemID: number) {
  const entry = detail.value?.problems?.find((item: DetailProblem) => item.problem.id === problemID)
  return entry ? `${problemDisplayCode(entry.problem)} · ${entry.problem.title}` : '未知题目'
}

watch(() => route.params.id, loadDetail)
watch(historyPageSize, clampHistoryPage)

function clampHistoryPage() {
  const maxPage = Math.max(1, Math.ceil(history.value.length / historyPageSize.value))
  if (historyPage.value > maxPage) historyPage.value = maxPage
  if (historyPage.value < 1) historyPage.value = 1
}

onMounted(loadDetail)
</script>

<style scoped>
.workbench {
  display: grid;
  gap: 14px;
}

.problem-strip {
  display: flex;
  align-items: stretch;
  flex-wrap: wrap;
  gap: 8px;
}

.problem-pick {
  display: grid;
  gap: 4px;
  min-width: 180px;
  max-width: 260px;
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

.editor-panel {
  display: grid;
  gap: 10px;
}

.test-data-panel {
  display: flex;
  justify-content: flex-end;
}


@media (max-width: 1100px) {
  .problem-pick {
    max-width: none;
    width: 100%;
  }
}
</style>
