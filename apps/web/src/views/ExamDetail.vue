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
        <el-button @click="leavePage">返回列表</el-button>
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
      </div>

      <div class="problem-strip">
        <button
          v-for="(entry, index) in detail.problems"
          :key="entry.problem.id"
          type="button"
          class="problem-pick"
          :class="{ active: activeProblem?.id === entry.problem.id }"
          @click="selectDetailProblem(entry)"
        >
          <strong>{{ problemLabel(entry, index) }} · {{ entry.problem.title }}</strong>
          <span>{{ entry.score }} 分 · {{ problemScoreText(entry.problem.id) }}</span>
          <small v-if="entry.problem.deleted_at" class="muted">已下架</small>
        </button>
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
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { client, sseUrl, type Problem, type Submission } from '../api/client'
import { formatDateTime, workStatusLabel } from '../features/assignments/assignmentMeta'
import { useAuthStore } from '../stores/auth'
import { useExamLockStore } from '../stores/examLock'

type DetailProblem = { problem: Problem; score: number; label?: string; problem_id: number }
type EditorState = { language: string; source: string; live: any }
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
const editorStates = reactive<Record<number, EditorState>>({})

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
    saveDraft(activeProblem.value.id, activeState.value.language, activeState.value.source)
    activeState.value.language = value
    activeState.value.source = loadDraft(activeProblem.value.id, value)
  }
})
const source = computed({
  get: () => activeState.value?.source || '',
  set: (value: string) => {
    if (!activeProblem.value || !activeState.value) return
    activeState.value.source = value
    saveDraft(activeProblem.value.id, activeState.value.language, value)
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
    activeEntry.value = detail.value.problems?.find((entry: DetailProblem) => entry.problem.id === activeProblem.value?.id) || detail.value.problems?.[0] || null
    if (activeProblem.value) ensureEditorState(activeProblem.value.id)
    await loadHistory()
  } catch (err: any) {
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
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    submitting.value = false
  }
}

function watchSubmission(id: number, problemID: number) {
  const es = new EventSource(sseUrl(`/submissions/${id}/events`))
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
  detail.value = (await client.get(`/exams/${detail.value.exam.id}`)).data
  activeEntry.value = detail.value.problems.find((entry: DetailProblem) => entry.problem.id === activeID) || detail.value.problems[0] || null
}

async function loadHistory() {
  if (!detail.value) return
  history.value = (await client.get('/submissions', { params: { exam_id: detail.value.exam.id } })).data
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

function leavePage() {
  if (examLocked.value) {
    ElMessage.warning(examLock.message)
    return
  }
  router.push('/exams')
}

function ensureEditorState(problemID: number) {
  if (!editorStates[problemID]) {
    editorStates[problemID] = { language: 'cpp', source: loadDraft(problemID, 'cpp'), live: null }
  }
  return editorStates[problemID]
}

function loadDraft(problemID: number, lang: string) {
  if (!detail.value) return defaultSource(lang)
  return localStorage.getItem(draftKey(problemID, lang)) || defaultSource(lang)
}

function saveDraft(problemID: number, lang: string, value: string) {
  if (!detail.value) return
  localStorage.setItem(draftKey(problemID, lang), value)
}

function draftKey(problemID: number, lang: string) {
  return `school-oj-draft:exam:${detail.value.exam.id}:${problemID}:${lang}`
}

function defaultSource(lang: string) {
  if (lang === 'python') return 'a, b = map(int, input().split())\nprint(a + b)\n'
  if (lang === 'java') return 'import java.util.*;\npublic class Main { public static void main(String[] args) { Scanner sc = new Scanner(System.in); long a = sc.nextLong(), b = sc.nextLong(); System.out.println(a + b); } }\n'
  if (lang === 'c') return '#include <stdio.h>\nint main(){ long long a,b; scanf("%lld%lld",&a,&b); printf("%lld\\n", a+b); return 0; }\n'
  return '#include <bits/stdc++.h>\nusing namespace std;\nint main(){ long long a,b; cin>>a>>b; cout<<a+b<<"\\n"; return 0; }\n'
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

function beforeUnload(event: BeforeUnloadEvent) {
  if (!examLocked.value) return
  event.preventDefault()
  event.returnValue = ''
}

watch(
  examLocked,
  (locked) => {
    if (locked) {
      examLock.lock(detail.value?.exam?.id)
      return
    }
    if (detail.value?.exam?.id === examLock.examId) examLock.unlock()
  },
  { immediate: true }
)
watch(() => route.params.id, loadDetail)

onBeforeRouteLeave(() => {
  if (examLocked.value) {
    ElMessage.warning(examLock.message)
    return false
  }
  if (detail.value?.exam?.id === examLock.examId) examLock.unlock()
})

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', beforeUnload)
})

onMounted(async () => {
  examLock.hydrate()
  window.addEventListener('beforeunload', beforeUnload)
  await loadDetail()
})
</script>

<style scoped>
.exam-workbench {
  display: grid;
  gap: 14px;
}

.exam-tabs,
.problem-strip {
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

.problem-pick {
  display: grid;
  gap: 4px;
  min-width: 180px;
  max-width: 260px;
  padding: 10px;
  border: 1px solid #d9dee8;
  border-radius: 8px;
  background: var(--surface);
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

@media (max-width: 760px) {
  .problem-select-row {
    align-items: stretch;
    flex-direction: column;
  }

  .problem-pick {
    max-width: none;
    width: 100%;
  }
}
</style>
