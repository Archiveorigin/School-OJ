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
        <el-button @click="router.push('/assignments')">返回列表</el-button>
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

      <section v-if="activeProblem" class="panel editor-panel">
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
        <div v-if="live" class="live"><StatusBadge :status="live.status" /> 分数 {{ live.score }}，{{ live.message }}</div>
      </section>

      <div class="panel history-panel">
        <div class="section-title"><h3>全部提交记录</h3></div>
        <el-table :data="history" size="small">
          <el-table-column label="题目" min-width="180">
            <template #default="{ row }">{{ problemTitle(row.problem_id) }}</template>
          </el-table-column>
          <el-table-column prop="language" label="语言" width="90" />
          <el-table-column label="状态" width="130"><template #default="{ row }"><StatusBadge :status="row.status" /></template></el-table-column>
          <el-table-column prop="score" label="原始分" width="90" />
          <el-table-column label="时间" min-width="170"><template #default="{ row }">{{ formatDateTime(row.created_at) }}</template></el-table-column>
        </el-table>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, sseUrl, type Problem, type Submission } from '../api/client'
import CodeEditor from '../components/CodeEditor.vue'
import ProblemStatementView from '../components/ProblemStatementView.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { formatDateTime, workStatusLabel } from '../features/assignments/assignmentMeta'
import { problemDisplayCode } from '../features/problems/problemMeta'

type DetailProblem = { problem: Problem; score: number; problem_id: number }
type EditorState = { language: string; source: string; live: any }

const route = useRoute()
const router = useRouter()
const detail = ref<any>(null)
const activeEntry = ref<DetailProblem | null>(null)
const activeProblem = computed(() => activeEntry.value?.problem || null)
const history = ref<Submission[]>([])
const submitting = ref(false)
const editorRef = ref<InstanceType<typeof CodeEditor> | null>(null)
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
    detail.value = (await client.get(`/assignments/${id}`)).data
    activeEntry.value = detail.value.problems?.[0] || null
    if (activeProblem.value) ensureEditorState(activeProblem.value.id)
    await loadHistory()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
    router.push('/assignments')
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
  detail.value = (await client.get(`/assignments/${detail.value.assignment.id}`)).data
  activeEntry.value = detail.value.problems.find((entry: DetailProblem) => entry.problem.id === activeID) || detail.value.problems[0] || null
}

async function loadHistory() {
  if (!detail.value) return
  history.value = (await client.get('/submissions', { params: { assignment_id: detail.value.assignment.id } })).data
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
  return `school-oj-draft:assignment:${detail.value.assignment.id}:${problemID}:${lang}`
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
  .problem-pick {
    max-width: none;
    width: 100%;
  }
}
</style>
