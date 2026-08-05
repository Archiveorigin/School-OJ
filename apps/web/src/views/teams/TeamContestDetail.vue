<template>
  <section class="contest-detail-view">
    <header class="contest-header">
      <div>
        <button type="button" class="back-button" @click="router.push(`/teams/${team.slug}/contests`)">返回比赛列表</button>
        <span class="eyebrow">TEAM CONTEST</span>
        <h1>{{ detail?.contest?.title || '团队比赛' }}</h1>
        <p v-if="detail">{{ contestTimeText }}</p>
      </div>
      <div v-if="detail" class="header-status">
        <el-tag :type="statusType(detail.contest.status)" size="large">{{ statusLabel(detail.contest.status) }}</el-tag>
        <span>{{ detail.problems.length }} 道题目</span>
        <el-button v-if="detail.can_organize" @click="addVisible = true">添加题目</el-button>
      </div>
    </header>

    <div v-if="detail" class="contest-body">
      <nav class="contest-tabs">
        <button type="button" :class="{ active: activeTab === 'overview' }" @click="activeTab = 'overview'">题目概览</button>
        <button type="button" :class="{ active: activeTab === 'problems' }" @click="activeTab = 'problems'">查看题目</button>
        <button type="button" :class="{ active: activeTab === 'records' }" @click="openRecords">提交记录</button>
        <button type="button" :class="{ active: activeTab === 'ranking' }" @click="openRanking">实时榜单</button>
      </nav>

      <ProblemOverview
        v-if="activeTab === 'overview'"
        :items="detail.problems"
        :active-problem-id="activeProblemID"
        @select="openContestProblem"
      />

      <section v-else-if="activeTab === 'problems'" class="problem-workspace">
        <div v-if="selectedLink" class="panel current-problem-bar">
          <div><strong>{{ selectedLink.label }} · {{ selectedLink.problem.title }}</strong><span>{{ selectedLink.problem.display_code }}</span></div>
          <div class="summary-actions">
            <StatusBadge v-if="selectedLink.submission_status" :status="selectedLink.submission_status" />
            <el-tag v-else type="info" effect="plain">未提交</el-tag>
            <el-button v-if="detail.can_organize" @click="addVisible = true">添加题目</el-button>
            <el-button v-if="detail.can_organize" type="danger" text @click="removeProblem">移出比赛</el-button>
            <el-button type="primary" :disabled="!detail.can_submit" @click="openSubmit">提交本题</el-button>
          </div>
        </div>

        <ProblemStatementView
          v-if="selectedLink"
          :problem="selectedLink.problem"
          :problem-number="selectedLink.label"
          :status-text="problemStatusText(selectedLink.submission_status)"
          :status-type="problemStatusType(selectedLink.submission_status)"
          :show-meta="false"
        />
        <el-empty v-else description="比赛暂未添加题目">
          <el-button v-if="detail.can_organize" type="primary" @click="addVisible = true">添加第一道题</el-button>
        </el-empty>
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
        <div class="ranking-heading"><div><span class="eyebrow">LIVE SCOREBOARD</span><h3>团队实时榜单</h3></div><el-button :loading="rankingLoading" @click="loadRanking">刷新</el-button></div>
        <el-table :data="ranking.rows || []" v-loading="rankingLoading">
          <el-table-column type="index" label="排名" width="76" />
          <el-table-column prop="name" label="成员" min-width="170" />
          <el-table-column prop="solved" label="通过" width="86" />
          <el-table-column prop="submission_count" label="提交" width="86" />
          <el-table-column v-for="problem in ranking.problems || []" :key="problem.problem_id" :label="problem.label" width="100" align="center">
            <template #default="{ row }"><StatusBadge v-if="rankingCell(row, problem.problem_id)?.status" :status="rankingCell(row, problem.problem_id).status" /><span v-else>-</span></template>
          </el-table-column>
          <el-table-column label="最后提交" min-width="180"><template #default="{ row }">{{ row.last_submission ? formatDateTime(row.last_submission) : '-' }}</template></el-table-column>
        </el-table>
      </section>
    </div>
    <el-skeleton v-else :rows="10" animated class="loading-panel" />

    <el-dialog v-model="submitVisible" :title="`提交 ${selectedLink?.label || ''} ${selectedLink?.problem?.title || ''}`" width="min(900px, calc(100vw - 24px))" destroy-on-close>
      <div class="submit-toolbar">
        <el-select v-model="language" style="width: 140px">
          <el-option label="C++17" value="cpp" /><el-option label="C" value="c" /><el-option label="Python" value="python" /><el-option label="Java" value="java" />
        </el-select>
        <span>代码仅计入本场团队比赛</span>
      </div>
      <CodeEditor v-model="source" :language="language" />
      <div v-if="liveStatus" class="live-status"><StatusBadge :status="liveStatus.status" /><span>{{ liveStatus.message || '评测中' }}</span></div>
      <template #footer><el-button @click="submitVisible = false">取消</el-button><el-button type="primary" :loading="submitting" @click="submitSolution">提交评测</el-button></template>
    </el-dialog>

    <el-dialog v-model="addVisible" title="添加比赛题目" width="480px">
      <p class="dialog-hint">输入公共题目或本团队私有题目的题号。</p>
      <el-input v-model="problemCode" placeholder="例如 T001" @keyup.enter="addProblem" />
      <template #footer><el-button @click="addVisible = false">取消</el-button><el-button type="primary" :loading="adding" @click="addProblem">添加</el-button></template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, openEventStream, type Submission, type Team } from '../../api/client'
import CodeEditor from '../../components/CodeEditor.vue'
import ProblemOverview from '../../components/ProblemOverview.vue'
import ProblemStatementView from '../../components/ProblemStatementView.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { formatDateTime } from '../../features/time'

const props = defineProps<{ team: Team }>()
const route = useRoute()
const router = useRouter()
const detail = ref<any>(null)
const activeTab = ref<'overview' | 'problems' | 'records' | 'ranking'>('overview')
const activeProblemID = ref<number>()
const submitVisible = ref(false)
const submitting = ref(false)
const language = ref('cpp')
const source = ref('')
const liveStatus = ref<any>(null)
const records = ref<Submission[]>([])
const recordsLoading = ref(false)
const ranking = ref<any>({ rows: [], problems: [] })
const rankingLoading = ref(false)
const addVisible = ref(false)
const adding = ref(false)
const problemCode = ref('')
const contestID = computed(() => Number(route.params.contestId))
const selectedLink = computed(() => detail.value?.problems?.find((item: any) => item.problem.id === activeProblemID.value) || null)
const contestTimeText = computed(() => {
  if (!detail.value?.contest) return ''
  const start = detail.value.contest.starts_at ? formatDateTime(detail.value.contest.starts_at) : '立即开始'
  const end = detail.value.contest.ends_at ? formatDateTime(detail.value.contest.ends_at) : '不限时'
  return `${start} — ${end}`
})

async function loadDetail() {
  try {
    const { data } = await client.get(`/teams/${props.team.id}/contests/${contestID.value}`)
    detail.value = data
    if (!data.problems?.some((item: any) => item.problem.id === activeProblemID.value)) activeProblemID.value = data.problems?.[0]?.problem.id
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
    await router.replace(`/teams/${props.team.slug}/contests`)
  }
}

function openSubmit() {
  if (!detail.value?.can_submit || !selectedLink.value) return
  source.value = ''
  liveStatus.value = null
  submitVisible.value = true
}

function openContestProblem(problemID: number) {
  activeProblemID.value = problemID
  activeTab.value = 'problems'
}

async function submitSolution() {
  if (!selectedLink.value || !source.value.trim()) {
    ElMessage.warning('请输入代码')
    return
  }
  submitting.value = true
  try {
    const { data } = await client.post(`/teams/${props.team.id}/contests/${contestID.value}/submissions`, {
      problem_id: selectedLink.value.problem.id,
      language: language.value,
      source_code: source.value
    })
    const stream = openEventStream(`/submissions/${data.id}/events`)
    stream.addEventListener('status', async (event) => {
      liveStatus.value = JSON.parse((event as MessageEvent).data)
      if (!['queued', 'running'].includes(liveStatus.value.status)) {
        stream.close()
        await loadDetail()
        await loadRecords()
      }
    })
    ElMessage.success('代码已提交评测')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    submitting.value = false
  }
}

async function addProblem() {
  if (!problemCode.value.trim()) return ElMessage.warning('请输入题目编号')
  adding.value = true
  try {
    await client.post(`/teams/${props.team.id}/contests/${contestID.value}/problems`, { problem_code: problemCode.value.trim() })
    problemCode.value = ''
    addVisible.value = false
    ElMessage.success('题目已加入比赛')
    await loadDetail()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    adding.value = false
  }
}

async function removeProblem() {
  if (!selectedLink.value) return
  try {
    await ElMessageBox.confirm(`确认将 ${selectedLink.value.problem.title} 移出比赛？`, '移出比赛', { type: 'warning' })
    await client.delete(`/teams/${props.team.id}/contests/${contestID.value}/problems/${selectedLink.value.problem.id}`)
    ElMessage.success('题目已移出比赛')
    await loadDetail()
  } catch (err: any) {
    if (err !== 'cancel' && err !== 'close') ElMessage.error(err.response?.data?.error || err.message)
  }
}

async function openRecords() { activeTab.value = 'records'; await loadRecords() }
async function loadRecords() {
  recordsLoading.value = true
  try { records.value = (await client.get(`/teams/${props.team.id}/contests/${contestID.value}/submissions`)).data || [] }
  finally { recordsLoading.value = false }
}
async function openRanking() { activeTab.value = 'ranking'; await loadRanking() }
async function loadRanking() {
  rankingLoading.value = true
  try { ranking.value = (await client.get(`/teams/${props.team.id}/contests/${contestID.value}/ranking`)).data || { rows: [], problems: [] } }
  finally { rankingLoading.value = false }
}

function problemLabel(problemID: number) { return detail.value?.problems?.find((item: any) => item.problem.id === problemID)?.label || '-' }
function rankingCell(row: any, problemID: number) { return row.problems?.find((item: any) => item.problem_id === problemID) }
function statusLabel(status: string) { return status === 'not_started' ? '未开始' : status === 'closed' ? '已结束' : '进行中' }
function statusType(status: string): 'success' | 'warning' | 'info' { return status === 'not_started' ? 'warning' : status === 'closed' ? 'info' : 'success' }
function problemStatusText(status?: string) { return !status ? '未提交' : status === 'accepted' ? '已通过' : ['queued', 'running'].includes(status) ? '评测中' : '未通过' }
function problemStatusType(status?: string): 'success' | 'warning' | 'info' | 'danger' { return status === 'accepted' ? 'success' : ['queued', 'running'].includes(status || '') ? 'warning' : status ? 'danger' : 'info' }

watch(contestID, loadDetail)
onMounted(loadDetail)
</script>

<style scoped>
.contest-detail-view { padding: 28px 34px 58px; }
.contest-header { display: flex; align-items: end; justify-content: space-between; gap: 24px; padding: 8px 0 24px; border-bottom: 1px solid var(--border); }
.contest-header h1 { margin: 8px 0 6px; font-size: 30px; }
.contest-header p { margin: 0; color: var(--muted); }
.back-button { display: block; margin: 0 0 18px; padding: 0; color: var(--muted); border: 0; background: transparent; cursor: pointer; }
.eyebrow { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .14em; }
.header-status { display: flex; align-items: center; gap: 12px; color: var(--muted); }
.contest-body { display: grid; gap: 18px; padding-top: 18px; }
.contest-tabs { display: flex; gap: 5px; border-bottom: 1px solid var(--border); }
.contest-tabs button { padding: 13px 22px; color: var(--muted); border: 0; border-bottom: 3px solid transparent; background: transparent; cursor: pointer; }
.contest-tabs button.active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 800; }
.problem-workspace { display: grid; gap: 14px; }
.current-problem-bar, .ranking-heading, .section-title { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.current-problem-bar > div:first-child { display: grid; gap: 4px; }
.current-problem-bar span, .submit-toolbar span { color: var(--muted); font-size: 12px; }
.summary-actions, .submit-toolbar, .live-status { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; }
.records-panel { padding: 18px; }
.ranking-panel { display: grid; gap: 16px; }
.ranking-heading h3 { margin: 5px 0 0; }
.loading-panel { padding: 30px; }
.submit-toolbar { margin-bottom: 12px; }
.live-status { margin-top: 12px; }
.dialog-hint { color: var(--muted); }
@media (max-width: 720px) {
  .contest-detail-view { padding: 20px 14px 44px; }
  .contest-header, .current-problem-bar, .ranking-heading { align-items: stretch; flex-direction: column; }
  .header-status, .summary-actions { justify-content: space-between; }
}
</style>
