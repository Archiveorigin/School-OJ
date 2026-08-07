<template>
  <section class="page contest-page">
    <div class="contest-container">
      <header class="contest-header">
        <div>
          <button type="button" class="back-button" @click="goBack">← 返回团队比赛</button>
          <span class="eyebrow">TEAM CONTEST</span>
          <h1>{{ detail?.contest?.title || '团队比赛' }}</h1>
          <p v-if="detail">{{ contestTimeText }}</p>
        </div>
        <div v-if="detail" class="header-status">
          <el-tag :type="statusType(detail.contest.status)" size="large">{{ statusLabel(detail.contest.status) }}</el-tag>
          <span>{{ detail.problems.length }} 道题目</span>
          <el-button v-if="detail.can_edit" @click="openEditContest">编辑设置</el-button>
          <el-button v-if="detail.can_edit" @click="addVisible = true">添加题目</el-button>
          <el-button v-if="detail.can_publish" type="success" @click="publishContest">发布并冻结</el-button>
        </div>
      </header>

      <div v-if="detail" class="contest-body">
        <nav class="contest-tabs" aria-label="比赛功能">
          <button v-for="tab in tabs" :key="tab.key" type="button" :class="{ active: activeTab === tab.key }" @click="openTab(tab.key)">{{ tab.label }}</button>
        </nav>

        <ProblemOverview v-if="activeTab === 'overview'" :items="detail.problems" :active-problem-id="selectedLink?.problem_id" @select="openContestProblem" />

        <section v-else-if="activeTab === 'problems'" class="problem-workspace">
          <div v-if="selectedLink" class="problem-actions">
            <div class="manage-actions">
              <el-button v-if="detail.can_edit" @click="addVisible = true">添加题目</el-button>
              <el-button v-if="detail.can_edit" type="danger" text @click="removeProblem">移出比赛</el-button>
            </div>
            <el-button type="primary" :disabled="!detail.can_submit" @click="openSubmit">提交代码</el-button>
          </div>
          <ProblemStatementView
            v-if="selectedLink"
            :problem="selectedLink.problem"
            :problem-number="selectedLink.label"
            :status-text="problemStatusText(selectedLink.submission_status)"
            :status-type="problemStatusType(selectedLink.submission_status)"
            :show-meta="false"
          />
          <div v-if="detail.problems.length > 1" class="problem-switcher">
            <button v-for="link in detail.problems" :key="link.problem_id" type="button" :class="{ active: link.problem_id === selectedLink?.problem_id }" @click="openContestProblem(link.problem_id)">{{ link.label }}</button>
          </div>
          <el-empty v-if="!selectedLink" description="比赛暂未添加题目"><el-button v-if="detail.can_edit" type="primary" @click="addVisible = true">添加第一道题</el-button></el-empty>
        </section>

        <section v-else-if="activeTab === 'records'" class="panel records-panel">
          <div class="section-title"><h3>我的比赛提交</h3><el-button :loading="recordsLoading" @click="loadRecords">刷新</el-button></div>
          <el-table :data="records">
            <el-table-column label="题号" width="90"><template #default="{ row }">{{ problemLabel(row.problem_id) }}</template></el-table-column>
            <el-table-column prop="problem_title" label="题目" min-width="190" />
            <el-table-column prop="language" label="语言" width="90" />
            <el-table-column label="状态" width="140"><template #default="{ row }"><StatusBadge :status="row.status" /></template></el-table-column>
            <el-table-column label="提交时间" min-width="180"><template #default="{ row }">{{ formatDateTime(row.created_at) }}</template></el-table-column>
          </el-table>
          <el-empty v-if="!recordsLoading && !records.length" description="还没有比赛提交" />
        </section>

        <section v-else class="ranking-panel">
          <LeaderboardBoard
            :data="scoreboardData"
            :loading="rankingLoading"
            :updated-at="rankingUpdatedAt"
            show-auto-refresh
            v-model:auto-refresh="rankingAutoRefresh"
            @refresh="loadRanking"
          />
        </section>
      </div>
      <el-skeleton v-else :rows="10" animated class="loading-panel" />

      <el-dialog v-model="submitVisible" :title="`提交 ${selectedLink?.label || ''} ${selectedLink?.problem?.title || ''}`" width="min(980px, calc(100vw - 24px))" destroy-on-close align-center>
        <SubmissionComposer ref="composerRef" v-model:language="language" v-model:source="source" :draft-context="draftContext" :status="liveStatus?.status" :message="liveStatus?.message" :submitting="submitting" scope-text="代码仅计入本场团队比赛" @submit="submitSolution" />
        <template #footer><el-button @click="submitVisible = false">关闭</el-button></template>
      </el-dialog>

      <el-dialog v-model="addVisible" title="添加比赛题目" width="480px">
        <p class="dialog-hint">输入公共题目或本团队私有题目的题号。</p>
        <el-input v-model="problemCode" placeholder="例如 T001" @keyup.enter="addProblem" />
        <template #footer><el-button @click="addVisible = false">取消</el-button><el-button type="primary" :loading="adding" @click="addProblem">添加</el-button></template>
      </el-dialog>

      <el-dialog v-model="editVisible" title="编辑比赛草稿" width="560px">
        <el-form label-position="top">
          <el-form-item label="比赛标题"><el-input v-model="editForm.title" maxlength="200" /></el-form-item>
          <el-form-item label="开始时间"><el-date-picker v-model="editForm.starts_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" /></el-form-item>
          <el-form-item label="比赛时长"><el-input-number v-model="editForm.duration_minutes" :min="15" :step="15" /></el-form-item>
          <el-form-item label="排名规则"><el-radio-group v-model="editForm.scoring_rule"><el-radio-button value="penalty">通过数 + 罚时</el-radio-button><el-radio-button value="score">通过数 + 总分</el-radio-button></el-radio-group></el-form-item>
          <el-form-item label="说明"><el-input v-model="editForm.description" type="textarea" :rows="4" /></el-form-item>
        </el-form>
        <template #footer><el-button @click="editVisible = false">取消</el-button><el-button type="primary" :loading="editing" @click="saveContest">保存</el-button></template>
      </el-dialog>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, getLatestSubmissions, openEventStream, type AuthenticatedEventSource, type Submission } from '../../api/client'
import LeaderboardBoard from '../../components/LeaderboardBoard.vue'
import ProblemOverview from '../../components/ProblemOverview.vue'
import ProblemStatementView from '../../components/ProblemStatementView.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import SubmissionComposer from '../../components/SubmissionComposer.vue'
import { formatDateTime } from '../../features/time'
import { adaptTeamContestRanking } from '../../features/leaderboard/adapters'
import { loadSubmissionDraft } from '../../features/submissions/drafts'
import { useAuthStore } from '../../stores/auth'

type ContestTab = 'overview' | 'problems' | 'records' | 'ranking'
const tabs: Array<{ key: ContestTab; label: string }> = [
  { key: 'overview', label: '题目概览' }, { key: 'problems', label: '查看题目' }, { key: 'records', label: '提交记录' }, { key: 'ranking', label: '实时榜单' }
]
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const detail = ref<any>(null)
const submitVisible = ref(false)
const submitting = ref(false)
const language = ref('cpp')
const source = ref('')
const liveStatus = ref<any>(null)
const composerRef = ref<{ clearDraft: () => void } | null>(null)
const records = ref<Submission[]>([])
const recordsLoading = ref(false)
const ranking = ref<any>({ rows: [], problems: [], scoring_rule: 'penalty' })
const rankingLoading = ref(false)
const rankingAutoRefresh = ref(true)
const rankingLoadedAt = ref<Date | null>(null)
const addVisible = ref(false)
const adding = ref(false)
const problemCode = ref('')
const editVisible = ref(false)
const editing = ref(false)
const editForm = reactive({ title: '', description: '', starts_at: '', duration_minutes: 120, scoring_rule: 'penalty' })
let submissionStream: AuthenticatedEventSource | null = null
let rankingRefreshTimer: number | undefined

const contestID = computed(() => Number(route.params.contestId))
const activeTab = computed<ContestTab>(() => {
  const value = route.hash.replace(/^#/, '') as ContestTab
  return tabs.some((tab) => tab.key === value) ? value : 'overview'
})
const selectedProblemID = computed(() => Number(route.query.problem) || detail.value?.problems?.[0]?.problem_id)
const selectedLink = computed(() => detail.value?.problems?.find((item: any) => item.problem_id === selectedProblemID.value) || detail.value?.problems?.[0] || null)
const draftContext = computed(() => ({ userId: auth.user?.id || 0, resourceType: 'contest' as const, resourceId: contestID.value, problemId: selectedLink.value?.problem_id || 0 }))
const contestTimeText = computed(() => {
  const contest = detail.value?.contest
  if (!contest) return ''
  return `${contest.starts_at ? formatDateTime(contest.starts_at) : '立即开始'} — ${contest.ends_at ? formatDateTime(contest.ends_at) : '不限时'}`
})
const scoreboardData = computed(() => adaptTeamContestRanking(ranking.value, detail.value || {}))
const rankingUpdatedAt = computed(() => rankingLoadedAt.value ? formatDateTime(rankingLoadedAt.value) : '')

async function loadDetail() {
  if (!contestID.value) return
  try {
    detail.value = (await client.get(`/contests/${contestID.value}`)).data
    await loadRecords()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
    await router.replace('/teams')
  }
}

function goBack() { router.push(detail.value?.team?.slug ? `/teams/${detail.value.team.slug}/contests` : '/teams') }
function openTab(tab: ContestTab) { router.push({ path: route.path, query: route.query, hash: `#${tab}` }) }
function openContestProblem(problemID: number) { router.push({ path: route.path, query: { ...route.query, problem: String(problemID) }, hash: '#problems' }) }

async function openSubmit() {
  if (!detail.value?.can_submit || !selectedLink.value) return
  const latest = (await getLatestSubmissions({ team_contest_id: contestID.value }, selectedLink.value.problem_id))[0]
  language.value = latest?.language || 'cpp'
  source.value = loadSubmissionDraft(draftContext.value, language.value) ?? latest?.source_code ?? ''
  liveStatus.value = latest || null
  submitVisible.value = true
}

async function submitSolution() {
  if (!selectedLink.value || !source.value.trim()) return ElMessage.warning('请输入代码')
  submitting.value = true
  try {
    const { data } = await client.post(`/contests/${contestID.value}/submissions`, { problem_id: selectedLink.value.problem_id, language: language.value, source_code: source.value })
    composerRef.value?.clearDraft()
    liveStatus.value = data
    submissionStream?.close()
    submissionStream = openEventStream(`/submissions/${data.id}/events`)
    submissionStream.addEventListener('status', async (event) => {
      liveStatus.value = JSON.parse((event as MessageEvent).data)
      if (!['queued', 'running'].includes(liveStatus.value.status)) {
        submissionStream?.close(); submissionStream = null
        await Promise.all([loadDetail(), loadRanking()])
      }
    })
    ElMessage.success('代码已提交评测')
  } catch (err: any) { ElMessage.error(err.response?.data?.error || err.message) }
  finally { submitting.value = false }
}

async function publishContest() {
  try {
    await ElMessageBox.confirm('发布后将冻结题目、开始时间和评分规则，且不能撤回。确认发布？', '发布比赛', { type: 'warning' })
    await client.post(`/contests/${contestID.value}/publish`)
    ElMessage.success('比赛已发布，题目与评分规则已冻结')
    await loadDetail()
  } catch (err: any) {
    if (err !== 'cancel' && err !== 'close') ElMessage.error(err.response?.data?.error || err.message)
  }
}

function openEditContest() {
  const contest = detail.value?.contest
  if (!contest) return
  Object.assign(editForm, { title: contest.title, description: contest.description || '', starts_at: contest.starts_at || '', duration_minutes: contest.duration_minutes, scoring_rule: contest.scoring_rule || 'penalty' })
  editVisible.value = true
}

async function saveContest() {
  if (!editForm.title.trim() || !editForm.starts_at) return ElMessage.warning('请填写标题和开始时间')
  editing.value = true
  try {
    await client.put(`/contests/${contestID.value}`, editForm)
    editVisible.value = false
    ElMessage.success('比赛草稿已保存')
    await loadDetail()
  } catch (err: any) { ElMessage.error(err.response?.data?.error || err.message) }
  finally { editing.value = false }
}

async function addProblem() {
  if (!problemCode.value.trim()) return ElMessage.warning('请输入题目编号')
  adding.value = true
  try {
    await client.post(`/contests/${contestID.value}/problems`, { problem_code: problemCode.value.trim() })
    problemCode.value = ''; addVisible.value = false; ElMessage.success('题目已加入比赛'); await loadDetail()
  } catch (err: any) { ElMessage.error(err.response?.data?.error || err.message) }
  finally { adding.value = false }
}

async function removeProblem() {
  if (!selectedLink.value) return
  try {
    await ElMessageBox.confirm(`确认将 ${selectedLink.value.problem.title} 移出比赛？`, '移出比赛', { type: 'warning' })
    await client.delete(`/contests/${contestID.value}/problems/${selectedLink.value.problem_id}`)
    ElMessage.success('题目已移出比赛'); await router.replace({ path: route.path, hash: '#problems' }); await loadDetail()
  } catch (err: any) { if (err !== 'cancel' && err !== 'close') ElMessage.error(err.response?.data?.error || err.message) }
}

async function loadRecords() {
  if (!contestID.value) return
  recordsLoading.value = true
  try { records.value = (await client.get(`/contests/${contestID.value}/submissions`)).data || [] }
  catch (err: any) { ElMessage.error(err.response?.data?.error || err.message) }
  finally { recordsLoading.value = false }
}
async function loadRanking() {
  if (!contestID.value || rankingLoading.value) return
  rankingLoading.value = true
  try {
    ranking.value = (await client.get(`/contests/${contestID.value}/ranking`)).data || { rows: [], problems: [], scoring_rule: detail.value?.contest?.scoring_rule || 'penalty' }
    rankingLoadedAt.value = new Date()
  }
  catch (err: any) { ElMessage.error(err.response?.data?.error || err.message) }
  finally { rankingLoading.value = false; scheduleRankingRefresh() }
}

function contestEnded() {
  const contest = ranking.value?.contest || detail.value?.contest
  if (!contest) return false
  if (contest.status === 'closed' || contest.state === 'closed') return true
  const end = contest.ends_at ? new Date(contest.ends_at).getTime() : Number.NaN
  return Number.isFinite(end) && Date.now() >= end
}

function clearRankingRefresh() {
  if (rankingRefreshTimer) window.clearTimeout(rankingRefreshTimer)
  rankingRefreshTimer = undefined
}

function scheduleRankingRefresh() {
  clearRankingRefresh()
  if (!rankingAutoRefresh.value || activeTab.value !== 'ranking' || contestEnded()) return
  rankingRefreshTimer = window.setTimeout(() => {
    if (contestEnded()) {
      clearRankingRefresh()
      return
    }
    void loadRanking()
  }, 5000)
}

function problemLabel(problemID: number) { return detail.value?.problems?.find((item: any) => item.problem_id === problemID)?.label || '-' }
function statusLabel(status: string) { return status === 'draft' ? '草稿' : status === 'published' ? '已发布' : status === 'closed' ? '已结束' : '进行中' }
function statusType(status: string): 'success' | 'warning' | 'info' { return status === 'draft' || status === 'published' ? 'warning' : status === 'closed' ? 'info' : 'success' }
function problemStatusText(status?: string) { return !status ? '未提交' : status === 'accepted' ? '已通过' : ['queued', 'running'].includes(status) ? '评测中' : '未通过' }
function problemStatusType(status?: string): 'success' | 'warning' | 'info' | 'danger' { return status === 'accepted' ? 'success' : ['queued', 'running'].includes(status || '') ? 'warning' : status ? 'danger' : 'info' }

watch(contestID, loadDetail)
watch(rankingAutoRefresh, scheduleRankingRefresh)
watch(activeTab, (tab) => {
  clearRankingRefresh()
  if (tab === 'ranking') void loadRanking()
  if (tab === 'records') void loadRecords()
})
onMounted(async () => { await loadDetail(); if (activeTab.value === 'ranking') await loadRanking() })
onBeforeUnmount(() => { submissionStream?.close(); clearRankingRefresh() })
</script>

<style scoped>
.contest-page { padding: 24px 20px 58px; }
.contest-container { width: min(1480px, 100%); margin: 0 auto; }
.contest-header { display: flex; align-items: end; justify-content: space-between; gap: 24px; padding: 8px 0 24px; border-bottom: 1px solid var(--border); }
.contest-header h1 { margin: 8px 0 6px; font-size: 30px; }
.contest-header p { margin: 0; color: var(--muted); }
.back-button { display: block; margin: 0 0 18px; padding: 0; color: var(--muted); border: 0; background: transparent; cursor: pointer; }
.eyebrow { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .14em; }
.header-status, .problem-actions, .section-title { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.contest-body { display: grid; gap: 18px; padding-top: 18px; }
.contest-tabs { display: flex; gap: 5px; overflow-x: auto; border-bottom: 1px solid var(--border); }
.contest-tabs button { padding: 13px 22px; white-space: nowrap; color: var(--muted); border: 0; border-bottom: 3px solid transparent; background: transparent; cursor: pointer; }
.contest-tabs button.active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 800; }
.problem-workspace { display: grid; gap: 14px; }
.problem-actions { justify-content: flex-end; }
.manage-actions { display: flex; gap: 8px; margin-right: auto; }
.problem-switcher { display: flex; justify-content: center; flex-wrap: wrap; gap: 8px; }
.problem-switcher button { min-width: 38px; padding: 8px 11px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface-strong); color: var(--text); cursor: pointer; }
.problem-switcher button.active { color: white; border-color: var(--accent); background: var(--accent); }
.records-panel { padding: 18px; }
.ranking-panel { display: grid; gap: 16px; min-width: 0; }
.loading-panel { padding: 30px; }
.dialog-hint { color: var(--muted); }
@media (max-width: 720px) {
  .contest-page { padding: 16px 12px 44px; }
  .contest-header, .problem-actions { align-items: stretch; flex-direction: column; }
  .header-status, .manage-actions { flex-wrap: wrap; }
  .problem-actions > .el-button { width: 100%; }
}
</style>
