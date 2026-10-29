<template>
  <section class="exam-page">
    <div v-if="detail" class="exam-shell">
      <EventWorkspaceHeader :title="detail.exam.title" :rule="detail.exam.scoring_rule" :status="lifecycleLabels[eventLifecycle(detail.exam)]"
        kind="考试" :base="base" :show-ranking="showRanking" :show-problems="detail.can_read_problems" :show-records="canViewRecords" :back-label="canManage ? '返回考试管理' : ''" @back="leavePage">
        <template #actions>
          <el-button v-if="detail.can_submit" @click="openEventTicket">考试反馈</el-button>
          <el-button v-if="!canManage && detail.can_submit" type="danger" :loading="finishing" @click="finishExam">交卷</el-button>
          <el-button v-else-if="canManage" @click="router.push(`/admin/exams/${examID}/activity`)">行为日志</el-button>
          <el-button v-else-if="detail.lifecycle==='running' && !detail.finished_at" type="primary" @click="initialize">进入考试</el-button>
        </template>
      </EventWorkspaceHeader>
      <main class="exam-content">
<EventAnnouncements v-if="detail.can_read_problems" kind="exam" :event-id="examID" :can-publish="canManage" />
        <div v-if="section === 'description'" class="description-layout">
          <section class="panel description-panel"><div class="section-heading"><h2>考试说明</h2><div class="event-management-actions"><el-button v-if="detail.can_read_problems" @click="attachmentsVisible = true">附件</el-button><el-button v-if="canManage" type="primary" @click="router.push(`/event-papers/exam/${examID}/edit`)">{{editable ? '组题' : '查看试卷与审核'}}</el-button><el-button v-if="editable" @click="router.push({path:`/event-papers/exam/${examID}/edit`,query:{tab:'settings'}})">考试设置</el-button></div></div><MarkdownRenderer :source="detail.exam.description || '暂无说明'" /></section>
          <aside class="panel exam-info"><h3>考试信息</h3><dl>
            <div><dt>开始时间</dt><dd>{{ formatDateTime(detail.exam.starts_at) }}</dd></div>
            <div><dt>结束时间</dt><dd>{{ formatDateTime(detail.exam.ends_at) }}</dd></div>
            <div><dt>计分规则</dt><dd>{{ scoringDescription(detail.exam.scoring_rule) }}</dd></div>
            <div><dt>评阅方式</dt><dd>{{ detail.manual_review ? '人工评阅' : '自动评测' }}</dd></div>
            <div><dt>封榜</dt><dd>{{ detail.exam.freeze_enabled ? `结束前 ${detail.exam.freeze_duration_minutes} 分钟` : '未启用' }}</dd></div>
          </dl></aside>
          <el-alert v-if="detail.lifecycle==='closed'" class="paper-full-width" :closable="false" type="info" title="本场已截止，现为只读状态。题目与附件仍可查看，等待管理员确认完结。" />
          <el-alert v-else-if="!detail.can_read_problems" class="paper-full-width" :closable="false" type="info" :title="detail.finished_at?'你已交卷，可查看本人提交记录。':'本场尚未开始，开始后可确认进入考试。'" />
        </div>
        <EventProblemList v-else-if="section === 'problems' && detail.can_read_problems" :items="items" :base="base" />
        <EventProblemPanel v-else-if="section === 'problem' && selected" :selected="selected" :items="items" :records="problemRecords" :base="base"
          :asset-base="`/exams/${examID}/problems/${selected.problem_id}/assets`" @refresh="loadHistory" @view-submission="viewSubmission">
          <template #actions>
            <el-button v-if="detail.can_submit" @click="openProblemTicket">工单修改</el-button>
            <el-button type="primary" :disabled="!detail.can_submit" @click="openSubmit">提交代码</el-button>
          </template>
        </EventProblemPanel>
        <ExamRanking v-else-if="section === 'scoreboard' && showRanking" :detail="detail" />
        <section v-else-if="section==='records' && canViewRecords" class="panel own-records"><h2>本人提交</h2><el-table :data="history.filter(row=>row.user_id===auth.user?.id)"><el-table-column prop="id" label="提交编号"/><el-table-column prop="status" label="状态"/><el-table-column label="时间"><template #default="{row}">{{ formatDateTime(row.created_at) }}</template></el-table-column><el-table-column label="操作"><template #default="{row}"><el-button link @click="viewSubmission(row.id)">查看提交</el-button></template></el-table-column></el-table><el-button @click="loadHistory">刷新提交</el-button></section>
        <el-result v-else title="此页面暂未开放"><template #extra><el-button @click="router.replace(`${base}/description`)">返回考试说明</el-button></template></el-result>
      </main>
      <el-dialog v-model="submitVisible" :title="`提交 ${selected?.label || ''}.${selected?.problem.title || ''}`" width="min(940px, calc(100vw - 24px))" destroy-on-close>
        <SubmissionComposer v-model:language="language" v-model:source="source" :draft-context="draftContext" :status="liveStatus" :submitting="submitting" :disabled="!detail.can_submit" @submit="submitSolution" />
      </el-dialog>
      <el-dialog v-model="attachmentsVisible" title="考试附件" width="min(760px, calc(100vw - 24px))" destroy-on-close><EventAttachments v-if="detail.can_read_problems" kind="exam" :event-id="examID" :editable="editable" :revision="detail.exam.review_revision" @changed="loadDetail" /></el-dialog>
      <TicketDialog v-model="ticketVisible" :context="{ kind: 'exam', id: examID, problemID: ticketProblemID }" />
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
import EventAnnouncements from '../components/events/EventAnnouncements.vue'
import {eventLifecycle,lifecycleLabels} from "../api/modules/events"
import { useSubmissionStreams } from "../features/submissions/live"
import { apiErrorMessage } from '../api/errors'
import EventAttachments from '../components/events/EventAttachments.vue'
import TicketDialog from '../components/tickets/TicketDialog.vue'

import { submissionLanguage } from '../features/submissions/presentation'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { generatedApi, getLatestSubmissions, type AuthenticatedEventSource, type Submission } from '../api/client'
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

const submissionStreams = useSubmissionStreams()
const route = useRoute(), router = useRouter(), auth = useAuthStore(), examLock = useExamLockStore()
const examID = computed(() => Number(route.params.id))
const base = computed(() => `/exams/${examID.value}`)
const detail = ref<any>(null), error = ref('')
const canManage = computed(() => Boolean(detail.value?.can_manage))
const attachmentsVisible = ref(false)
const editable = computed(() => canManage.value && ['draft','rejected'].includes(detail.value?.exam.review_status))
const canViewRecords = computed(() => !canManage.value && ['running','closed'].includes(detail.value?.lifecycle))
const showRanking = computed(() => detail.value?.exam.scoring_rule !== 'oi' && (canManage.value || detail.value?.ranking_visible))
const active = computed(() => auth.role === 'student' && Boolean(detail.value) && detail.value.can_submit)
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
    await loadDetail()
    if (!detail.value) return
    if (auth.role === 'student' && eventLifecycle(detail.value.exam)==='running' && !detail.value.finished_at) {
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
    if (route.query.problem_id && ['problems', 'submit'].includes(String(section.value))) {
      const target = items.value.find(item => item.problem_id === Number(route.query.problem_id))
      if (target) await router.replace(problemPath(base.value, target.label))
    }
    if (section.value === 'scoreboard' && !showRanking.value) await router.replace(`${base.value}/description`)
  } catch (err: any) {
    error.value = apiErrorMessage(err)
    if (auth.role === 'student' && [403, 404].includes(err.response?.status) && err.response?.data?.code !== 'EXAM_SCOPE_REQUIRED') {
      const state = await examLock.syncActiveExam().catch(() => null)
      if (state && !state.active) { await exitExam(error.value); return }
    }
  } finally { initializing = false }
}
async function loadDetail() {
  try { detail.value = (await generatedApi.getExam({ path: { id: examID.value } })).data }
  catch (err: any) {
    if (err.response?.data?.code === 'EVENT_COMPLETED') { examLock.unlock(); detail.value = null; await router.replace(`/competitions/exam/${examID.value}`) }
    throw err
  }
  if (deadline) window.clearTimeout(deadline)
  if (detail.value.lifecycle === 'running' && detail.value.exam.ends_at) {
    const serverNow = new Date(detail.value.now).getTime()
    const delay = new Date(detail.value.exam.ends_at).getTime() - serverNow
    deadline = window.setTimeout(() => {
      if (detail.value) {
        detail.value.can_submit = false;
        detail.value.closed = true; submitVisible.value = false;
      }
      void checkState()
    }, Math.min(Math.max(100, delay + 100), 2147483647))
  }
}
async function loadHistory() {
  if (!detail.value || (!canManage.value && !canViewRecords.value)) { history.value = []; latest.value = []; return }
  try {
    const [rows, recent] = await Promise.all([generatedApi.listSubmissions<any>({ query: { exam_id: examID.value } }), getLatestSubmissions({ exam_id: examID.value })])
    history.value = rows.data; latest.value = recent as Submission[]
  } catch (err: any) { ElMessage.error(apiErrorMessage(err, '提交记录加载失败')) }
}
async function checkState() {
  if (initializing || !detail.value) return
  try {
    if (auth.role === 'student' && (active.value || examLock.locked)) {
      const state = await examLock.syncActiveExam()
      if (!state.active) {
        await flush(); examLock.unlock(); stream?.close(); submitVisible.value = false;
        await loadDetail(); await loadHistory();
        if (!detail.value.can_read_problems && !['description', 'records'].includes(String(section.value))) await router.replace(`${base.value}/description`)
        return
      }
      if (state.examId !== examID.value) { await router.replace(`/exams/${state.examId}/description`); return }
    }
    await loadDetail()
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
    const { data } = await generatedApi.createSubmission<any>({ body: { problem_id: selected.value.problem_id, exam_id: examID.value, language: submissionLanguage(language.value), source_code: source.value } })
    clearSubmissionDraft(draftContext.value, language.value)
    liveStatus.value = data.status
    stream?.close()
    if (['queued', 'running'].includes(data.status)) {
      stream = submissionStreams.watch(data.id, state => { liveStatus.value = state.status }, loadHistory)
    }
    await loadHistory()
  } catch (err: any) { ElMessage.error(apiErrorMessage(err, '提交失败')); await checkState() }
  finally { submitting.value = false }
}
async function viewSubmission(id: number) {
  try { submissionDetail.value = (await generatedApi.getSubmission({ path: { id } })).data; submissionVisible.value = true }
  catch (err: any) { ElMessage.error(apiErrorMessage(err, '提交加载失败')) }
}
const ticketVisible = ref(false), ticketProblemID = ref<number | undefined>()
function openEventTicket() { ticketProblemID.value = undefined; ticketVisible.value = true }
function openProblemTicket() { if (selected.value) { ticketProblemID.value = selected.value.problem_id; ticketVisible.value = true } }
async function finishExam() {
  try { await ElMessageBox.confirm('交卷后不能再次进入本场考试，也不能继续提交。确认交卷？', '交卷', { type: 'warning', confirmButtonText: '确认交卷', cancelButtonText: '取消' }) }
  catch { return }
  finishing.value = true
  try {
    await flush()
    await generatedApi.finishExam({ path: { id: examID.value } })
    await exitExam('已交卷')
  } catch (err: any) { ElMessage.error(apiErrorMessage(err, '交卷失败')) }
  finally { finishing.value = false }
}
watch(() => route.params.label, () => { submitVisible.value = false; submissionVisible.value = false; stream?.close() })
watch(examID, () => { detail.value = null; history.value = []; latest.value = []; void initialize() })
onMounted(() => { void initialize(); poller = window.setInterval(checkState, 15000) })
onBeforeUnmount(() => { if (poller) window.clearInterval(poller); if (deadline) window.clearTimeout(deadline); stream?.close() })
</script>

<style scoped>
.section-heading { display:flex; flex-wrap:wrap; align-items:center; gap:12px; margin-bottom:18px; }.section-heading h2 { margin:0; }.event-management-actions { display:flex; flex-wrap:wrap; gap:8px; }.event-management-actions :deep(.el-button + .el-button) { margin-left:0; }
.paper-full-width { grid-column: 1 / -1; }
.exam-page { min-height: 100%; padding: 24px 20px 58px; background: color-mix(in srgb, var(--surface) 74%, var(--color-canvas)); }
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
