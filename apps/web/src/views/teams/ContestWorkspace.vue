<template>
  <section class="contest-page">
    <div v-if="detail" class="contest-shell">
      <EventWorkspaceHeader :title="detail.contest.title" :subtitle="detail.team.name" :rule="detail.contest.scoring_rule" :status="statusLabel(detail.contest.status)"
        kind="比赛" :base="`/contest/${contestID}`" :show-ranking="true" back-label="返回团队比赛" @back="goBack" />

      <main class="contest-content">
        <template v-if="section === 'description'">
          <div class="two-column">
            <section class="panel description-panel">
              <div class="section-heading">
                <h2>比赛说明</h2>
                <div v-if="detail.can_edit">
                  <el-button
                    v-if="detail.can_publish"
                    type="success"
                    :loading="publishing"
                    @click="publishContest"
                    >发布比赛</el-button
                  ><el-button @click="editVisible = true">编辑设置</el-button
                  ><el-button @click="addVisible = true">添加题目</el-button>
                </div>
              </div>
              <div class="description-copy">
                {{ detail.contest.description || '本场比赛暂未填写说明。' }}
              </div>
            </section>
            <aside class="panel meta-panel">
              <h3>比赛信息</h3>
              <dl>
                <div>
                  <dt>开始时间</dt>
                  <dd>{{ dateText(detail.contest.starts_at) }}</dd>
                </div>
                <div>
                  <dt>结束时间</dt>
                  <dd>{{ dateText(detail.contest.ends_at) }}</dd>
                </div>
                <div>
                  <dt>比赛时长</dt>
                  <dd>{{ detail.contest.duration_minutes }} 分钟</dd>
                </div>
                <div>
                  <dt>计分规则</dt>
                  <dd>{{ scoringDescription(detail.contest.scoring_rule) }}</dd>
                </div>
                <div>
                  <dt>封榜</dt>
                  <dd>
                    {{
                      detail.contest.freeze_enabled
                        ? `结束前 ${detail.contest.freeze_duration_minutes} 分钟`
                        : '未启用'
                    }}
                  </dd>
                </div>
              </dl>
            </aside>
          </div>
        </template>

        <EventProblemList v-else-if="section === 'problems'" :items="problemItems" :base="`/contest/${contestID}`">
          <template #actions><el-button v-if="detail.can_edit" type="primary" @click="addVisible = true">添加题目</el-button></template>
        </EventProblemList>
        <EventProblemPanel v-else-if="section === 'problem' && selectedLink" :selected="{ ...selectedLink, score: 100 }" :items="problemItems"
          :records="problemRecords" :base="`/contest/${contestID}`" @refresh="loadRecords" @view-submission="viewSubmission">
          <template #actions><el-button type="primary" :disabled="!detail.can_submit" @click="openSubmit">提交代码</el-button></template>
        </EventProblemPanel>

        <template v-else-if="section === 'scoreboard' && detail.contest.scoring_rule !== 'oi'">
          <section class="contest-scoreboard">
            <LeaderboardBoard
              :data="scoreboardData"
              :loading="rankingLoading"
              :error="rankingError"
              empty-text="暂无比赛记录"
              :updated-at="rankingUpdatedAt"
              show-auto-refresh
              v-model:auto-refresh="autoRefresh"
              @refresh="loadRanking"
            />
          </section>
        </template>
        <el-result v-else title="题目不存在或页面未开放"><template #extra><el-button @click="router.replace(`/contest/${contestID}/problems`)">题目列表</el-button></template></el-result>
      </main>

      <SubmissionDetailDialog v-model="submissionVisible" :detail="submissionDetail" />
      <el-dialog
        v-model="submitVisible"
        :title="`提交 ${selectedLink?.label || ''} · ${selectedLink?.problem.title || ''}`"
        width="min(940px, calc(100vw - 24px))"
        destroy-on-close
        ><SubmissionComposer
          ref="composer"
          v-model:language="language"
          v-model:source="source"
          :draft-context="draftContext"
          :status="liveStatus?.status"
          :submitting="submitting"
          @submit="submitSolution"
        /><template #footer
          ><el-button @click="submitVisible = false">关闭</el-button></template
        ></el-dialog
      >
      <el-dialog v-model="addVisible" title="添加比赛题目" width="460px"
        ><p class="muted">输入公共题目或本团队私有题目的题号。</p>
        <el-input
          v-model="problemCode"
          placeholder="例如 T001"
          @keyup.enter="addProblem"
        /><template #footer
          ><el-button @click="addVisible = false">取消</el-button
          ><el-button type="primary" :loading="adding" @click="addProblem"
            >添加</el-button
          ></template
        ></el-dialog
      >
      <el-dialog v-model="editVisible" title="比赛设置" width="min(620px, calc(100vw - 24px))"
        ><el-form label-position="top"
          ><el-form-item label="比赛标题"><el-input v-model="editForm.title" /></el-form-item
          ><el-form-item label="开始时间"
            ><el-date-picker
              v-model="editForm.starts_at"
              type="datetime"
              value-format="YYYY-MM-DDTHH:mm:ssZ" /></el-form-item
          ><el-form-item label="时长（分钟）"
            ><el-input-number v-model="editForm.duration_minutes" :min="15" /></el-form-item
          ><el-form-item label="赛制"
            ><el-segmented
              v-model="editForm.scoring_rule"
              :options="scoringOptions" /></el-form-item
          ><el-form-item label="封榜"
            ><el-switch
              v-model="editForm.freeze_enabled"
              :disabled="editForm.scoring_rule === 'oi'" /><el-input-number
              v-if="editForm.freeze_enabled"
              v-model="editForm.freeze_duration_minutes"
              :min="1"
              :max="Math.max(1, editForm.duration_minutes - 1)" /></el-form-item
          ><el-form-item label="说明"
            ><el-input
              v-model="editForm.description"
              type="textarea"
              :rows="4" /></el-form-item></el-form
        ><template #footer
          ><el-button @click="editVisible = false">取消</el-button
          ><el-button type="primary" :loading="editing" @click="saveContest"
            >保存</el-button
          ></template
        ></el-dialog
      >
    </div>
    <el-skeleton v-else :rows="12" animated class="loading" />
  </section>
</template>

<script setup lang="ts">
import { submissionLanguage } from '../../features/submissions/presentation'
import { ElMessage } from 'element-plus'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { generatedApi, getLatestSubmissions,
  openEventStream,
  type AuthenticatedEventSource,
  type ScoringRule,
  type Submission
} from '../../api/client'
import LeaderboardBoard from '../../components/LeaderboardBoard.vue'
import EventWorkspaceHeader from '../../components/EventWorkspaceHeader.vue'
import EventProblemList from '../../components/EventProblemList.vue'
import EventProblemPanel from '../../components/EventProblemPanel.vue'
import SubmissionDetailDialog from '../../components/SubmissionDetailDialog.vue'
import { scoringDescription } from '../../features/events/workspace'
import SubmissionComposer from '../../components/SubmissionComposer.vue'
import { adaptTeamContestRanking } from '../../features/leaderboard/adapters'
import { loadSubmissionDraft } from '../../features/submissions/drafts'
import { formatDateTime } from '../../features/time'
import { useAuthStore } from '../../stores/auth'
const route = useRoute(),
  router = useRouter(),
  auth = useAuthStore()
const detail = ref<any>(null),
  ranking = ref<any>({ rows: [], problems: [] }),
  records = ref<Submission[]>([]),
  rankingLoading = ref(false),
  rankingError = ref(''),
  rankingLoadedAt = ref<Date | null>(null),
  autoRefresh = ref(true),
  submitVisible = ref(false),
  submitting = ref(false),
  language = ref('cpp'),
  source = ref(''),
  liveStatus = ref<any>(null),
  composer = ref<any>(null),
  addVisible = ref(false),
  adding = ref(false),
  problemCode = ref(''),
  editVisible = ref(false),
  editing = ref(false),
  publishing = ref(false)
let stream: AuthenticatedEventSource | null = null
let rankingTimer: number | undefined
const scoringOptions = [
  { label: 'OI', value: 'oi' },
  { label: 'IOI', value: 'ioi' },
  { label: 'ACM', value: 'acm' }
]
const editForm = reactive({
  title: '',
  description: '',
  starts_at: '',
  duration_minutes: 120,
  scoring_rule: 'acm' as ScoringRule,
  freeze_enabled: false,
  freeze_duration_minutes: 60,
  gold_award_percent: 10,
  silver_award_percent: 10,
  bronze_award_percent: 10
})
const contestID = computed(() => Number(route.params.contestId))
const section = computed(() =>
  route.name === 'contest-description'
    ? 'description'
    : route.name === 'contest-problems'
      ? 'problems'
      : route.name === 'contest-problem'
        ? 'problem'
        : 'scoreboard'
)
const selectedLink = computed(
  () =>
    detail.value?.problems?.find(
      (i: any) => String(i.label).toLowerCase() === String(route.params.label || '').toLowerCase()
    ) || null
)
const problemItems = computed(() => (detail.value?.problems || []).map((link: any) => ({ ...link, score: 100 })))
const submissionVisible = ref(false), submissionDetail = ref<any>(null)
async function viewSubmission(id: number) {
  try { submissionDetail.value = (await generatedApi.getSubmission<any>({ path: { "id": id } })).data; submissionVisible.value = true }
  catch (err: any) { ElMessage.error(err.response?.data?.error || '提交加载失败') }
}
const problemRecords = computed(() =>
  records.value.filter((i) => i.problem_id === selectedLink.value?.problem_id)
)
const draftContext = computed(() => ({
  userId: auth.user?.id || 0,
  resourceType: 'contest' as const,
  resourceId: contestID.value,
  problemId: selectedLink.value?.problem_id || 0
}))
const scoreboardData = computed(() => {
  const data = adaptTeamContestRanking(ranking.value, {
    contest: detail.value?.contest
  })
  return data
})
const rankingUpdatedAt = computed(() =>
  rankingLoadedAt.value ? formatDateTime(rankingLoadedAt.value) : ''
)
async function loadDetail() {
  try {
    detail.value = (await generatedApi.getTeamContestViaContestsContestId<any>({ path: { "contest_id": contestID.value } })).data
    if (detail.value.contest.scoring_rule === 'oi' && section.value === 'scoreboard')
      await router.replace(`/contest/${contestID.value}/problems`)
    Object.assign(editForm, { ...detail.value.contest })
    if (route.query.manage === 'edit' && detail.value.can_edit) {
      editVisible.value = true
      await router.replace({ path: route.path, query: {} })
    }
    await loadRecords()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || e.message)
    await router.replace('/teams')
  }
}
async function loadRecords() {
  if (!contestID.value) return
  try {
    records.value = (await generatedApi.listTeamContestSubmissionsViaContestsContestIdSubmissions<any>({ path: { "contest_id": contestID.value } })).data || []
  } catch {
    records.value = []
  }
}
async function loadRanking() {
  if (detail.value?.contest.scoring_rule === 'oi' || rankingLoading.value) return
  rankingLoading.value = true
  rankingError.value = ''
  try {
    ranking.value = (await generatedApi.teamContestRankingViaContestsContestIdRanking<any>({ path: { "contest_id": contestID.value } })).data || {
      rows: [],
      problems: []
    }
    rankingLoadedAt.value = new Date()
  } catch (e: any) {
    ranking.value = { rows: [], problems: [] }
    rankingError.value = e.response?.data?.error || e.message
  } finally {
    rankingLoading.value = false
    scheduleRankingRefresh()
  }
}
function goBack() {
  void router.push(
    detail.value?.team?.slug ? `/teams/${detail.value.team.slug}/contests` : '/teams'
  )
}
function dateText(v?: string) {
  return v ? formatDateTime(v) : '待定'
}
function statusLabel(v: string) {
  return v === 'draft'
    ? '草稿'
    : v === 'published'
      ? '待开始'
      : v === 'running'
        ? '进行中'
        : '已结束'
}
function clearRankingTimer() {
  if (rankingTimer) window.clearTimeout(rankingTimer)
  rankingTimer = undefined
}
function scheduleRankingRefresh() {
  clearRankingTimer()
  if (
    !autoRefresh.value ||
    section.value !== 'scoreboard' ||
    ['ended', 'closed'].includes(detail.value?.contest?.state)
  )
    return
  rankingTimer = window.setTimeout(() => void loadRanking(), 5000)
}
async function openSubmit() {
  if (!selectedLink.value) return
  const latest = (
    await getLatestSubmissions({ team_contest_id: contestID.value }, selectedLink.value.problem_id)
  )[0]
  language.value = latest?.language || 'cpp'
  source.value =
    loadSubmissionDraft(draftContext.value, language.value) ?? latest?.source_code ?? ''
  liveStatus.value = latest || null
  submitVisible.value = true
}
async function submitSolution() {
  if (!source.value.trim()) return ElMessage.warning('请输入代码')
  submitting.value = true
  try {
    const { data } = await generatedApi.createTeamContestSubmissionViaContestsContestIdSubmissions<any>({ path: { "contest_id": contestID.value }, body: {
      problem_id: selectedLink.value.problem_id,
      language: submissionLanguage(language.value),
      source_code: source.value
    } })
    composer.value?.clearDraft()
    liveStatus.value = data
    stream?.close()
    stream = openEventStream(`/submissions/${data.id}/events`)
    stream.addEventListener('status', async (event: MessageEvent<string>) => {
      liveStatus.value = JSON.parse(event.data)
      if (!['queued', 'running'].includes(liveStatus.value.status)) {
        stream?.close()
        stream = null
        await Promise.all([loadDetail(), loadRanking()])
      }
    })
    ElMessage.success('代码已提交')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    submitting.value = false
  }
}
async function addProblem() {
  if (!problemCode.value.trim()) return
  adding.value = true
  try {
    await generatedApi.addTeamContestProblemViaContestsContestIdProblems<any>({ path: { "contest_id": contestID.value }, body: {
      problem_code: problemCode.value.trim()
    } })
    problemCode.value = ''
    addVisible.value = false
    await loadDetail()
    ElMessage.success('题目已加入比赛')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    adding.value = false
  }
}
async function saveContest() {
  if (editForm.scoring_rule === 'oi') editForm.freeze_enabled = false
  editing.value = true
  try {
    await generatedApi.updateTeamContestViaContestsContestId<any>({ path: { "contest_id": contestID.value }, body: editForm })
    editVisible.value = false
    await loadDetail()
    ElMessage.success('设置已保存')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    editing.value = false
  }
}
async function publishContest() {
  publishing.value = true
  try {
    await generatedApi.publishTeamContestViaContestsContestIdPublish<any>({ path: { "contest_id": contestID.value } })
    await loadDetail()
    ElMessage.success('比赛已发布')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    publishing.value = false
  }
}
watch(autoRefresh, scheduleRankingRefresh)
watch(
  () => route.fullPath,
  async () => {
    clearRankingTimer()
    if (!detail.value || Number(route.params.contestId) !== detail.value.contest.id)
      await loadDetail()
    if (section.value === 'scoreboard') await loadRanking()
  }
)
onMounted(async () => {
  await loadDetail()
  if (section.value === 'scoreboard') await loadRanking()
})
onBeforeUnmount(() => {
  stream?.close()
  clearRankingTimer()
})
</script>

<style scoped>
.contest-page { min-height: 100%; padding: 24px 20px 58px; background: color-mix(in srgb, var(--surface) 74%, var(--color-canvas)); }
.contest-shell { width: min(1320px, 100%); margin: 0 auto; }
.contest-content { padding-top: 16px; }
.contest-scoreboard { min-width: 0; overflow: hidden; }
.two-column { display: grid; grid-template-columns: minmax(0, 1fr) 320px; gap: 16px; }
.description-panel, .meta-panel { padding: 20px; }
.section-heading { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 18px; }
.section-heading h2 { margin: 0; }
.description-copy { min-height: 240px; margin-top: 18px; padding-top: 18px; white-space: pre-wrap; line-height: 1.8; border-top: 1px solid var(--border); }
.meta-panel h3 { margin-top: 0; }
.meta-panel dl div { display: grid; grid-template-columns: 88px 1fr; gap: 8px; padding: 12px 0; border-top: 1px solid var(--border); }
.meta-panel dt { color: var(--muted); }
.meta-panel dd { margin: 0; text-align: right; }
.loading { max-width: 1320px; margin: 0 auto; }
@media (max-width: 880px) { .two-column { grid-template-columns: minmax(0, 1fr); } }
@media (max-width: 600px) { .contest-page { padding: 16px 10px 36px; } }
</style>
