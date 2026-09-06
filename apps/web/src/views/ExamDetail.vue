<template>
  <section class="exam-page">
    <div v-if="detail" class="exam-shell">
      <EventWorkspaceHeader :title="detail.exam.title" :rule="detail.exam.scoring_rule" :status="detail.closed ? '已结束' : detail.not_started ? '未开始' : '进行中'"
        kind="考试" :base="base" :show-ranking="showRanking" :back-label="canManage ? '返回考试管理' : ''" @back="leavePage">
        <template #actions>
          <el-button v-if="!canManage" type="danger" :loading="finishing" @click="finishExam">交卷</el-button>
          <el-button v-else @click="router.push(`/admin/exams/${examID}/activity`)">行为日志</el-button>
        </template>
      </EventWorkspaceHeader>
      <main class="exam-content">
        <div v-if="section === 'description'" class="description-layout">
          <section class="panel description-panel"><h2>考试说明</h2><MarkdownRenderer :source="detail.exam.description || '暂无说明'" /></section>
          <aside class="panel exam-info"><h3>考试信息</h3><dl>
            <div><dt>开始时间</dt><dd>{{ formatDateTime(detail.exam.starts_at) }}</dd></div>
            <div><dt>结束时间</dt><dd>{{ formatDateTime(detail.exam.ends_at) }}</dd></div>
            <div><dt>计分规则</dt><dd>{{ scoringDescription(detail.exam.scoring_rule) }}</dd></div>
            <div><dt>评阅方式</dt><dd>{{ detail.manual_review ? '人工评阅' : '自动评测' }}</dd></div>
            <div><dt>封榜</dt><dd>{{ detail.exam.freeze_enabled ? `结束前 ${detail.exam.freeze_duration_minutes} 分钟` : '未启用' }}</dd></div>
          </dl></aside>
        </div>
        <EventProblemList v-else-if="section === 'problems'" :items="items" :base="base" />
        <EventProblemPanel v-else-if="section === 'problem' && selected" :selected="selected" :items="items" :records="problemRecords" :base="base"
          :asset-base="`/exams/${examID}/problems/${selected.problem_id}/assets`" @refresh="loadHistory" @view-submission="viewSubmission">
          <template #actions>
            <el-button v-if="canManage" @click="openProblemTicket">工单修改</el-button>
            <el-button type="primary" :disabled="!detail.can_submit" @click="openSubmit">提交代码</el-button>
          </template>
        </EventProblemPanel>
        <ExamRanking v-else-if="section === 'scoreboard' && showRanking" :detail="detail" />
        <el-result v-else title="题目不存在或页面未开放"><template #extra><el-button @click="router.replace(`${base}/problems`)">题目列表</el-button></template></el-result>
      </main>
      <el-dialog v-model="submitVisible" :title="`提交 ${selected?.label || ''}.${selected?.problem.title || ''}`" width="min(940px, calc(100vw - 24px))" destroy-on-close>
        <SubmissionComposer v-model:language="language" v-model:source="source" :draft-context="draftContext" :status="liveStatus" :submitting="submitting" :disabled="!detail.can_submit" @submit="submitSolution" />
      </el-dialog>
      <SubmissionDetailDialog v-model="submissionVisible" :detail="submissionDetail" />
    </div>
    <el-result v-else-if="error" title="无法加载考试" :sub-title="error"><template #extra><el-button @click="initialize">重试</el-button><el-button v-if="!examLock.locked" @click="leavePage">返回课程</el-button></template></el-result>
    <el-skeleton v-else :rows="12" animated />
    <el-dialog v-model="warning" title="考试环境警告" width="min(440px, calc(100vw - 24px))" :close-on-click-modal="false" :close-on-press-escape="false" :show-close="false">
      <p>检测到离开考试窗口或使用受限快捷键，操作已记录。请继续在本场考试内作答。</p>
      <template #footer><el-button type="primary" @click="warning = false">返回作答</el-button></template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, generatedApi, getLatestSubmissions, openEventStream, type AuthenticatedEventSource, type Submission } from '../api/client'
import EventWorkspaceHeader from '../components/EventWorkspaceHeader.vue'
import EventProblemList from '../components/EventProblemList.vue'
import EventProblemPanel from '../components/EventProblemPanel.vue'
import MarkdownRenderer from '../components/MarkdownRenderer.vue'
import SubmissionComposer from '../components/SubmissionComposer.vue'
import SubmissionDetailDialog from '../components/SubmissionDetailDialog.vue'
import ExamRanking from './exam/ExamRanking.vue'
import { examProblemItems, problemPath } from '../features/events/workspace'
import { scoringDescription } from '../features/events/workspace'
import { formatDateTime } from '../features/time'
import { clearSubmissionDraft, loadSubmissionDraft } from '../features/submissions/drafts'
import { useExamEnvironment } from '../features/exams/useExamEnvironment'
import { useAuthStore } from '../stores/auth'
import { useExamLockStore } from '../stores/examLock'

const route = useRoute(), router = useRouter(), auth = useAuthStore(), examLock = useExamLockStore()
const examID = computed(() => Number(route.params.id))
const base = computed(() => `/exams/${examID.value}`)
const detail = ref<any>(null), error = ref('')
const canManage = computed(() => Boolean(detail.value?.can_manage))
const showRanking = computed(() => detail.value?.exam.scoring_rule !== 'oi' && (canManage.value || detail.value?.ranking_visible))
const active = computed(() => auth.role === 'student' && Boolean(detail.value) && !detail.value.closed && !detail.value.finished_at)
const { warning, flush } = useExamEnvironment(active, computed(() => auth.user?.id || 0), examID)
const history = ref<Submission[]>([]), latest = ref<Submission[]>([])
const items = computed(() => examProblemItems(detail.value?.problems || [], latest.value))
const section = computed(() => route.params.label ? 'problem' : route.path.split('/').pop())
const selected = computed(() => items.value.find(item => item.label === route.params.label))
const problemRecords = computed(() => history.value.filter(row => row.problem_id === selected.value?.problem_id && row.user_id === auth.user?.id))
const submitVisible = ref(false), submissionVisible = ref(false), submissionDetail = ref<any>(null)
const language = ref('cpp'), source = ref(''), liveStatus = ref(''), submitting = ref(false), finishing = ref(false)
const draftContext = computed(() => ({ userId: auth.user?.id || 0, resourceType: 'exam' as const, resourceId: examID.value, problemId: selected.value?.problem_id || 0 }))
let poller: number | undefined, deadline: number | undefined, stream: AuthenticatedEventSource | undefined
let initializing = false
function listPath() { return canManage.value ? '/admin/exams' : detail.value?.exam.course_id ? `/my/courses/${detail.value.exam.course_id}/exams` : '/my/courses' }
function leavePage() { void router.push(listPath()) }
async function exitExam(message: string) {
  const path = listPath()
  await flush()
  if (detail.value) detail.value.closed = true
  examLock.unlock()
  stream?.close()
  ElMessage.info(message)
  await router.replace(path)
}
async function initialize() {
  if (initializing) return
  initializing = true; error.value = ''
  try {
    if (auth.role === 'student') {
      const state = await examLock.syncActiveExam()
      if (!state.active) {
        try { await ElMessageBox.confirm('确认进入后，仅可访问本场考试。离开窗口会记录警告，交卷后不能再次进入。', '进入考试', { confirmButtonText: '确认进入', cancelButtonText: '取消', type: 'warning' }) }
        catch { leavePage(); return }
        await generatedApi.enterExam({ path: { id: examID.value } })
        examLock.lock(examID.value)
      }
    }
    await loadDetail()
    await loadHistory()
    if (route.query.problem_id && ['problems', 'submit', 'records'].includes(String(section.value))) {
      const target = items.value.find(item => item.problem_id === Number(route.query.problem_id))
      if (target) await router.replace(problemPath(base.value, target.label))
    }
    if (section.value === 'scoreboard' && !showRanking.value) await router.replace(`${base.value}/description`)
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message
    if (auth.role === 'student' && [403, 404].includes(err.response?.status) && err.response?.data?.code !== 'EXAM_SCOPE_REQUIRED') {
      const state = await examLock.syncActiveExam().catch(() => null)
      if (state && !state.active) { await exitExam(error.value); return }
    }
  } finally { initializing = false }
}
async function loadDetail() {
  detail.value = (await generatedApi.getExam({ path: { id: examID.value } })).data
  if (deadline) window.clearTimeout(deadline)
  if (active.value && detail.value.exam.ends_at) {
    const serverNow = new Date(detail.value.now).getTime()
    const delay = new Date(detail.value.exam.ends_at).getTime() - serverNow
    deadline = window.setTimeout(checkState, Math.min(Math.max(100, delay + 100), 2147483647))
  }
}
async function loadHistory() {
  if (!detail.value) return
  try {
    const [rows, recent] = await Promise.all([client.get('/submissions', { params: { exam_id: examID.value } }), getLatestSubmissions({ exam_id: examID.value })])
    history.value = rows.data; latest.value = recent as Submission[]
  } catch (err: any) { ElMessage.error(err.response?.data?.error || '提交记录加载失败') }
}
async function checkState() {
  if (!active.value) return
  try {
    const state = await examLock.syncActiveExam()
    if (!state.active) { await exitExam('考试已交卷或截止'); return }
    if (state.examId !== examID.value) { await router.replace(`/exams/${state.examId}/description`); return }
    await loadHistory()
  } catch { /* Server remains authoritative; a failed check must not unlock. */ }
}
function openSubmit() {
  if (!selected.value) return
  const recent = latest.value.find(row => row.problem_id === selected.value?.problem_id)
  language.value = recent?.language || 'cpp'
  source.value = loadSubmissionDraft(draftContext.value, language.value) ?? recent?.source_code ?? ''
  liveStatus.value = recent?.status || ''
  submitVisible.value = true
}
async function submitSolution() {
  if (!selected.value || !source.value.trim()) { ElMessage.warning('请输入代码'); return }
  submitting.value = true
  try {
    const { data } = await client.post('/submissions', { problem_id: selected.value.problem_id, exam_id: examID.value, language: language.value, source_code: source.value })
    clearSubmissionDraft(draftContext.value, language.value)
    liveStatus.value = data.status
    stream?.close()
    if (['queued', 'running'].includes(data.status)) {
      stream = openEventStream(`/submissions/${data.id}/events`)
      stream.addEventListener('status', event => {
        const state = JSON.parse((event as MessageEvent).data)
        liveStatus.value = state.status
        if (!['queued', 'running'].includes(state.status)) { stream?.close(); void loadHistory() }
      })
    }
    await loadHistory()
  } catch (err: any) { ElMessage.error(err.response?.data?.error || '提交失败'); await checkState() }
  finally { submitting.value = false }
}
async function viewSubmission(id: number) {
  try { submissionDetail.value = (await generatedApi.getSubmission({ path: { id } })).data; submissionVisible.value = true }
  catch (err: any) { ElMessage.error(err.response?.data?.error || '提交加载失败') }
}
function openProblemTicket() {
  if (selected.value) void router.push({ path: '/problem-changes/new', query: { action: 'replace', problem_id: selected.value.problem_id, target_scope: 'public' } })
}
async function finishExam() {
  try { await ElMessageBox.confirm('交卷后不能再次进入本场考试，也不能继续提交。确认交卷？', '交卷', { type: 'warning', confirmButtonText: '确认交卷', cancelButtonText: '取消' }) }
  catch { return }
  finishing.value = true
  try {
    await flush()
    await generatedApi.finishExam({ path: { id: examID.value } })
    await exitExam('已交卷')
  } catch (err: any) { ElMessage.error(err.response?.data?.error || '交卷失败') }
  finally { finishing.value = false }
}
watch(() => route.params.label, () => { submitVisible.value = false; submissionVisible.value = false; stream?.close() })
watch(examID, () => { detail.value = null; history.value = []; latest.value = []; void initialize() })
onMounted(() => { void initialize(); poller = window.setInterval(checkState, 15000) })
onBeforeUnmount(() => { if (poller) window.clearInterval(poller); if (deadline) window.clearTimeout(deadline); stream?.close() })
</script>

<style scoped>
.exam-page { min-height: 100%; padding: 24px 20px 58px; background: color-mix(in srgb, var(--surface) 74%, #edf2f7); }
.exam-shell { width: min(1320px, 100%); margin: 0 auto; }
.exam-content { padding-top: 16px; min-width: 0; }
.description-layout { display: grid; grid-template-columns: minmax(0, 1fr) 320px; gap: 16px; }
.description-panel, .exam-info { padding: 20px; min-width: 0; }
.description-panel h2, .exam-info h3 { margin: 0 0 18px; }
.description-panel { min-height: 300px; }
.exam-info dl > div { display: grid; grid-template-columns: 80px 1fr; gap: 8px; padding: 12px 0; border-top: 1px solid var(--border); }
.exam-info dt { color: var(--muted); }
.exam-info dd { margin: 0; text-align: right; overflow-wrap: anywhere; }
@media (max-width: 880px) { .description-layout { grid-template-columns: minmax(0, 1fr); } }
@media (max-width: 600px) { .exam-page { padding: 16px 10px 36px; } }
</style>
