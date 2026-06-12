<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h2>{{ detail?.exam?.title || '考试' }}</h2>
        <p v-if="detail" class="muted">结束时间：{{ formatDateTime(detail.exam.ends_at) }}</p>
      </div>
      <div class="toolbar">
        <el-tag v-if="detail?.closed" type="info">已结束</el-tag>
        <el-tag v-else-if="detail?.not_started" type="warning">未开始</el-tag>
        <el-tag v-else-if="detail" type="success">进行中</el-tag>
        <el-tag v-if="detail?.manual_review" type="warning">人工阅卷</el-tag>
        <el-tag v-if="detail?.all_submitted" type="success">已提交全部题目</el-tag>
        <el-tag v-if="detail">{{ workStatusLabel(detail.work_status) }}</el-tag>
        <strong v-if="detail">{{ scoreSummary }}</strong>
        <el-button
          v-if="!canManage"
          type="danger"
          :loading="finishing"
          :disabled="detail?.not_started || detail?.finished_at"
          @click="finishExam"
        >
          结束考试
        </el-button>
        <el-button v-if="canManage" @click="leavePage">返回列表</el-button>
      </div>
    </div>

    <div v-if="detail" class="exam-workbench">
      <div class="exam-tabs">
        <el-button :type="tabType('problems')" @click="goExamTab('problems')">查看题目</el-button>
        <el-button :type="tabType('submit')" @click="goExamTab('submit')">提交代码</el-button>
        <el-button :type="tabType('records')" @click="goExamTab('records')">提交记录</el-button>
      </div>

      <div class="problem-select-row">
        <span class="muted">题目选择</span>
        <el-select v-model="activeProblemID" filterable class="problem-select" placeholder="选择题目">
          <el-option
            v-for="entry in detail.problems"
            :key="entry.problem.id"
            :label="problemOptionLabel(entry)"
            :value="entry.problem.id"
          />
        </el-select>
        <el-button v-if="canManage && activeProblem" type="primary" plain @click="openProblemEditor">修改题目</el-button>
      </div>

      <router-view v-slot="{ Component }">
        <component
          :is="Component"
          :detail="detail"
          :active-entry="activeEntry"
          :active-problem="activeProblem"
          :history="history"
          :language="language"
          :source="source"
          :live="live"
          :submitting="submitting"
          :can-manage="canManage"
          @update:language="language = $event"
          @update:source="source = $event"
          @submit="submitSolution"
          @refresh-history="loadHistory"
        />
      </router-view>
      <ProblemEditDialog v-model="problemEditorVisible" :problem="activeProblem" @saved="handleProblemSaved" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { client, sseUrl, type Problem, type Submission } from '../api/client'
import ProblemEditDialog from '../components/ProblemEditDialog.vue'
import { formatDateTime, workStatusLabel } from '../features/assignments/assignmentMeta'
import { useAuthStore } from '../stores/auth'
import { useExamLockStore } from '../stores/examLock'

type DetailProblem = { problem: Problem; score: number; label?: string; problem_id: number }
type EditorState = { language: string; source: string; live: any; dirty: boolean }
type ExamTab = 'problems' | 'submit' | 'records'

const auth = useAuthStore()
const examLock = useExamLockStore()
const route = useRoute()
const router = useRouter()
const canManage = computed(() => auth.role !== 'student')
const detail = ref<any>(null)
const activeEntry = ref<DetailProblem | null>(null)
const activeProblem = computed(() => activeEntry.value?.problem || null)
const history = ref<Submission[]>([])
const submitting = ref(false)
const finishing = ref(false)
const problemEditorVisible = ref(false)
const editorStates = reactive<Record<number, EditorState>>({})
let deadlineTimer: ReturnType<typeof window.setTimeout> | null = null
let deadlinePoller: ReturnType<typeof window.setInterval> | null = null
let forceLeavingExam = false
const liveStreams = new Set<EventSource>()

const examID = computed(() => Number(route.params.id))
const examLocked = computed(() => {
  return !canManage.value && Boolean(detail.value) && !detail.value.closed && !detail.value.not_started && !detail.value.finished_at
})
const activeState = computed(() => {
  if (!activeProblem.value) return null
  return ensureEditorState(activeProblem.value.id)
})
const activeProblemID = computed({
  get: () => activeProblem.value?.id,
  set: (value: number | undefined) => {
    const entry = detail.value?.problems?.find((item: DetailProblem) => item.problem.id === value)
    if (entry) selectDetailProblem(entry)
  }
})
const language = computed({
  get: () => activeState.value?.language || 'cpp',
  set: (value: string) => {
    if (!activeProblem.value || !activeState.value) return
    activeState.value.language = value
    activeState.value.source = preferredSource(activeProblem.value.id, value)
    activeState.value.dirty = false
  }
})
const source = computed({
  get: () => activeState.value?.source || '',
  set: (value: string) => {
    if (!activeProblem.value || !activeState.value) return
    activeState.value.source = value
    activeState.value.dirty = true
  }
})
const live = computed(() => activeState.value?.live)
const activeTab = computed<ExamTab>(() => {
  const value = route.path.split('/').pop()
  if (value === 'submit' || value === 'records') return value
  return 'problems'
})
const scoreSummary = computed(() => {
  if (!detail.value) return ''
  if (detail.value.score_ready) return `${detail.value.total_score} / ${detail.value.max_score}`
  return detail.value.work_status === 'submitted' ? '待评分' : '-'
})

async function loadDetail() {
  if (!examID.value) return
  try {
    detail.value = (await client.get(`/exams/${examID.value}`)).data
    if (!canManage.value && detail.value.closed) {
      exitClosedExam()
      return
    }
    activeEntry.value = detail.value.problems?.find((entry: DetailProblem) => entry.problem.id === activeProblem.value?.id) || detail.value.problems?.[0] || null
    if (activeProblem.value) ensureEditorState(activeProblem.value.id)
    await loadHistory()
    scheduleDeadlineCheck()
  } catch (err: any) {
    if (handleInvalidExamError(err)) return
    if (err.response?.data?.finished_at) {
      examLock.unlock()
      ElMessage.warning('考试已结束，不能再次进入')
    } else {
      ElMessage.error(err.response?.data?.error || err.message)
    }
    router.push('/exams')
  }
}

function selectDetailProblem(entry: DetailProblem) {
  activeEntry.value = entry
  ensureEditorState(entry.problem.id)
}

function goExamTab(tab: ExamTab) {
  router.push(`/exams/${examID.value}/${tab}`)
}

function openProblemEditor() {
  if (!activeProblem.value) return
  problemEditorVisible.value = true
}

async function handleProblemSaved() {
  await refreshDetail()
  ElMessage.info('历史提交不会自动重判，需要时可在提交记录中手动重判')
}

function tabType(tab: ExamTab) {
  return activeTab.value === tab ? 'primary' : ''
}

async function submitSolution() {
  if (!activeProblem.value || !detail.value || !activeState.value) return
  const problemID = activeProblem.value.id
  submitting.value = true
  try {
    const { data } = await client.post('/submissions', {
      problem_id: problemID,
      exam_id: detail.value.exam.id,
      language: activeState.value.language,
      source_code: activeState.value.source
    })
    watchSubmission(data.id, problemID)
    await loadHistory()
  } catch (err: any) {
    if (handleInvalidExamError(err)) return
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    submitting.value = false
  }
}

function watchSubmission(id: number, problemID: number) {
  const es = new EventSource(sseUrl(`/submissions/${id}/events`))
  liveStreams.add(es)
  es.addEventListener('status', async (event) => {
    ensureEditorState(problemID).live = JSON.parse((event as MessageEvent).data)
    if (!['queued', 'running'].includes(ensureEditorState(problemID).live.status)) {
      closeLiveStream(es)
      await refreshDetail()
      await loadHistory()
    }
  })
}

async function refreshDetail() {
  if (!detail.value) return
  const activeID = activeProblem.value?.id
  try {
    detail.value = (await client.get(`/exams/${detail.value.exam.id}`)).data
  } catch (err: any) {
    if (handleInvalidExamError(err)) return
    throw err
  }
  if (!canManage.value && detail.value.closed) {
    exitClosedExam()
    return
  }
  activeEntry.value = detail.value.problems.find((entry: DetailProblem) => entry.problem.id === activeID) || detail.value.problems[0] || null
  scheduleDeadlineCheck()
}

async function loadHistory() {
  if (!detail.value) return
  history.value = (await client.get('/submissions', { params: { exam_id: detail.value.exam.id } })).data
  hydrateEditorStatesFromHistory()
}

async function finishExam() {
  if (!detail.value) return
  try {
    await ElMessageBox.confirm('结束后不能再次进入本场考试，也不能继续提交。确认结束？', '结束考试', { type: 'warning' })
  } catch {
    return
  }
  finishing.value = true
  try {
    const { data } = await client.post(`/exams/${detail.value.exam.id}/finish`)
    detail.value.finished_at = data.finished_at
    examLock.unlock()
    ElMessage.success('考试已结束')
    router.push('/exams')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    finishing.value = false
  }
}

async function leavePage() {
  if (deadlineReached()) {
    await handleDeadlineReached()
    return
  }
  router.push('/exams')
}

function ensureEditorState(problemID: number) {
  if (!editorStates[problemID]) {
    const submission = preferredSubmission(problemID)
    const language = submission?.language || 'cpp'
    editorStates[problemID] = { language, source: submission?.source_code || '', live: null, dirty: false }
  }
  return editorStates[problemID]
}

function hydrateEditorStatesFromHistory() {
  for (const entry of detail.value?.problems || []) {
    const state = editorStates[entry.problem.id]
    if (!state || state.dirty) continue
    const submission = preferredSubmission(entry.problem.id, state.language) || preferredSubmission(entry.problem.id)
    if (submission) {
      state.language = submission.language
      state.source = submission.source_code || ''
      continue
    }
    state.source = preferredSource(entry.problem.id, state.language)
  }
}

function preferredSubmission(problemID: number, language?: string) {
  const items = history.value.filter((item) => item.problem_id === problemID && (!language || item.language === language))
  return items.find((item) => item.status === 'accepted') || items[0] || null
}

function preferredSource(problemID: number, language: string) {
  return preferredSubmission(problemID, language)?.source_code || ''
}

function problemScoreText(problemID: number) {
  const item = scoreForProblem(problemID)
  if (!item?.submission_id) return '未提交'
  if (detail.value?.manual_review && !item.score_ready) return '待评分'
  if (!item.score_ready) return '计算中'
  return `${item.best_score} / ${item.score}`
}

function scoreForProblem(problemID: number) {
  return detail.value?.problem_scores?.find((item: any) => item.problem.id === problemID)
}

function problemOptionLabel(entry: DetailProblem) {
  return `${problemLabel(entry)} · ${entry.problem.title} · ${entry.score} 分 · ${problemScoreText(entry.problem.id)}`
}

function problemLabel(entry: DetailProblem, index?: number) {
  if (entry.label?.trim()) return entry.label.trim()
  const position = typeof index === 'number' ? index : detail.value?.problems?.findIndex((item: DetailProblem) => item.problem.id === entry.problem.id)
  if (typeof position === 'number' && position >= 0) return defaultProblemLabel(position)
  return defaultProblemLabel(0)
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

function scheduleDeadlineCheck() {
  clearDeadlineTimers()
  if (canManage.value || !detail.value?.exam?.ends_at || detail.value.closed || detail.value.finished_at) return
  const deadline = new Date(detail.value.exam.ends_at).getTime()
  const delay = deadline - Date.now()
  if (delay <= 0) {
    void handleDeadlineReached()
    return
  }
  deadlineTimer = window.setTimeout(handleDeadlineReached, Math.min(delay + 1000, 2147483647))
  deadlinePoller = window.setInterval(() => {
    if (deadlineReached()) void handleDeadlineReached()
  }, 30000)
}

function clearDeadlineTimers() {
  if (deadlineTimer) {
    window.clearTimeout(deadlineTimer)
    deadlineTimer = null
  }
  if (deadlinePoller) {
    window.clearInterval(deadlinePoller)
    deadlinePoller = null
  }
}

function deadlineReached() {
  if (!detail.value?.exam?.ends_at) return false
  return Date.now() >= new Date(detail.value.exam.ends_at).getTime()
}

async function handleDeadlineReached() {
  if (canManage.value || !detail.value) return
  clearDeadlineTimers()
  try {
    const { data } = await client.get(`/exams/${detail.value.exam.id}`)
    detail.value = data
  } catch (err: any) {
    if (handleInvalidExamError(err)) return
    // If refresh fails at the cutoff moment, unlock locally so the student can leave.
  }
  exitClosedExam()
}

function exitClosedExam() {
  clearDeadlineTimers()
  examLock.unlock()
  if (detail.value) detail.value.closed = true
  ElMessage.warning('考试已截止，已自动退出')
  router.replace('/exams')
}

function handleInvalidExamError(err: any) {
  if (canManage.value || !isInvalidExamError(err)) return false
  const message = String(err?.response?.data?.error || '').toLowerCase()
  if (message.includes('has not started')) {
    forceExitExam('考试未开始，不能进入')
  } else {
    forceExitExam('考试已删除或已失效，已自动退出')
  }
  return true
}

function isInvalidExamError(err: any) {
  const status = err?.response?.status
  if (status === 404) return true
  if (status !== 403 || err?.response?.data?.finished_at) return false
  const message = String(err?.response?.data?.error || '').toLowerCase()
  return message.includes('forbidden') || message.includes('not available') || message.includes('not found') || message.includes('has not started')
}

function forceExitExam(message: string) {
  forceLeavingExam = true
  clearExamRuntimeState(currentExamID())
  ElMessage.warning(message)
  router.replace('/exams')
}

function currentExamID() {
  return detail.value?.exam?.id || examID.value || examLock.examId
}

function clearExamRuntimeState(id?: number) {
  clearDeadlineTimers()
  closeLiveStreams()
  examLock.unlock()
  detail.value = null
  activeEntry.value = null
  history.value = []
  submitting.value = false
  finishing.value = false
  for (const key of Object.keys(editorStates)) delete editorStates[Number(key)]
  clearExamDraftCache(id)
}

function closeLiveStream(stream: EventSource) {
  stream.close()
  liveStreams.delete(stream)
}

function closeLiveStreams() {
  for (const stream of liveStreams) stream.close()
  liveStreams.clear()
}

function clearExamDraftCache(id?: number) {
  if (!id) return
  const prefix = `school-oj-draft:exam:${id}:`
  for (let index = localStorage.length - 1; index >= 0; index -= 1) {
    const key = localStorage.key(index)
    if (key?.startsWith(prefix)) localStorage.removeItem(key)
  }
}

watch(
  examLocked,
  (locked) => {
    if (locked) {
      examLock.lock(detail.value?.exam?.id, undefined, detail.value?.exam?.title)
      return
    }
    if (detail.value?.exam?.id === examLock.examId) examLock.unlock()
  },
  { immediate: true }
)
watch(() => route.params.id, loadDetail)

onBeforeRouteLeave(() => {
  if (forceLeavingExam) return true
  if (!examLocked.value && detail.value?.exam?.id === examLock.examId) examLock.unlock()
  return true
})

onBeforeUnmount(() => {
  clearDeadlineTimers()
  closeLiveStreams()
})

onMounted(async () => {
  examLock.hydrate()
  await loadDetail()
})
</script>

<style scoped>
.exam-workbench {
  display: grid;
  gap: 14px;
}

.exam-tabs {
  display: flex;
  align-items: stretch;
  flex-wrap: wrap;
  gap: 8px;
}

.problem-select-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.problem-select {
  width: min(520px, 100%);
}

@media (max-width: 760px) {
  .problem-select-row {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
