<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h2>{{ detail?.exam?.title || '考试' }}</h2>
        <p v-if="detail" class="muted">结束时间：{{ detail.exam.ends_at || '-' }}</p>
      </div>
      <div class="toolbar">
        <el-tag v-if="detail?.closed" type="info">已结束</el-tag>
        <el-tag v-else-if="detail?.not_started" type="warning">未开始</el-tag>
        <el-tag v-else-if="detail" type="success">进行中</el-tag>
        <el-tag v-if="detail?.manual_review" type="warning">人工阅卷</el-tag>
        <el-tag v-if="detail?.all_submitted" type="success">已提交全部题目</el-tag>
        <el-tag v-if="detail">{{ workStatusLabel(detail.work_status) }}</el-tag>
        <strong v-if="detail">{{ detail.score_ready ? `${detail.total_score} / ${detail.max_score}` : (detail.work_status === 'submitted' ? '待评分' : '-') }}</strong>
        <el-button v-if="!canManage" type="danger" :loading="finishing" :disabled="detail?.not_started || detail?.finished_at" @click="finishExam">结束考试</el-button>
        <el-button @click="leavePage">返回列表</el-button>
      </div>
    </div>

    <div v-if="detail" class="workbench">
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
            <small v-if="entry.problem.deleted_at" class="muted">已下架</small>
          </button>
        </aside>

        <main v-if="activeProblem" class="statement-panel">
          <div class="panel-title"><h3>{{ activeProblem.title }}</h3><span>{{ activeEntry?.score }} 分</span></div>
          <p class="muted">{{ activeProblem.time_limit_ms }} ms / {{ activeProblem.memory_limit_mb }} MB / {{ activeProblem.output_limit_kb }} KB</p>
          <MarkdownRenderer :source="activeProblem.statement" :problem-id="activeProblem.id" />
        </main>

        <section v-if="activeProblem" class="editor-panel">
          <div class="toolbar editor-toolbar">
            <el-select v-model="language" style="width: 130px">
              <el-option label="C++17" value="cpp" />
              <el-option label="C" value="c" />
              <el-option label="Python" value="python" />
              <el-option label="Java" value="java" />
            </el-select>
            <el-button @click="formatSource">自动格式化</el-button>
            <el-button type="primary" :loading="submitting" :disabled="!detail.can_submit" @click="submitSolution">提交</el-button>
          </div>
          <CodeEditor ref="editorRef" v-model="source" :language="language" />
          <div v-if="live" class="live"><StatusBadge :status="live.status" /> {{ live.status === 'pending_review' ? '等待教师评分' : `分数 ${live.score}，${live.message}` }}</div>
        </section>
      </div>

      <div class="panel history-panel">
        <div class="section-title"><h3>全部提交记录</h3></div>
        <el-table :data="history" size="small">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column label="题目" min-width="180">
            <template #default="{ row }">{{ problemTitle(row.problem_id) }}</template>
          </el-table-column>
          <el-table-column prop="language" label="语言" width="90" />
          <el-table-column label="状态" width="130"><template #default="{ row }"><StatusBadge :status="row.status" /></template></el-table-column>
          <el-table-column prop="score" label="参考分" width="90" />
          <el-table-column label="最终分" width="90"><template #default="{ row }">{{ row.manual_score ?? '-' }}</template></el-table-column>
          <el-table-column label="时间" min-width="160"><template #default="{ row }">{{ row.created_at }}</template></el-table-column>
        </el-table>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { client, sseUrl, type Problem, type Submission } from '../api/client'
import CodeEditor from '../components/CodeEditor.vue'
import MarkdownRenderer from '../components/MarkdownRenderer.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useAuthStore } from '../stores/auth'
import { useExamLockStore } from '../stores/examLock'

type DetailProblem = { problem: Problem; score: number; problem_id: number }
type EditorState = { language: string; source: string; live: any }

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
const editorRef = ref<InstanceType<typeof CodeEditor> | null>(null)
const editorStates = reactive<Record<number, EditorState>>({})

const examLocked = computed(() => {
  return !canManage.value && Boolean(detail.value) && !detail.value.closed && !detail.value.not_started && !detail.value.finished_at
})
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

async function loadDetail() {
  const id = Number(route.params.id)
  if (!id) return
  try {
    detail.value = (await client.get(`/exams/${id}`)).data
    activeEntry.value = detail.value.problems?.[0] || null
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
  ensureEditorState(entry.problem.id).live = null
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

function formatSource() {
  editorRef.value?.format()
}

function scoreForProblem(problemID: number) {
  return detail.value?.problem_scores?.find((item: any) => item.problem.id === problemID)
}

function problemScoreText(problemID: number) {
  const item = scoreForProblem(problemID)
  if (!item?.submission_id) return '未提交'
  if (detail.value?.manual_review && !item.score_ready) return '待评分'
  if (!item.score_ready) return '计算中'
  return `${item.best_score} / ${item.score}`
}

function problemTitle(problemID: number) {
  return detail.value?.problems?.find((entry: DetailProblem) => entry.problem.id === problemID)?.problem.title || `#${problemID}`
}

function workStatusLabel(status: string) {
  if (status === 'submitted') return '已提交'
  if (status === 'unsubmitted') return '未提交'
  return '未尝试'
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
.workbench {
  display: grid;
  gap: 14px;
}

.coding-grid {
  display: grid;
  grid-template-columns: 220px minmax(280px, 0.75fr) minmax(520px, 1.45fr);
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
