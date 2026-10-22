<template>
  <section class="contest-page">
    <div v-if="detail" class="contest-shell">
      <EventWorkspaceHeader
        :title="detail.contest.title"
        :subtitle="detail.team.name"
        :rule="detail.contest.scoring_rule"
        :status="lifecycleLabels[eventLifecycle(detail.contest)]"
        kind="比赛"
        :base="`/contest/${contestID}`"
        :show-ranking="detail.can_read_problems"
        :show-problems="detail.can_read_problems"
        :show-records="canViewRecords"
        back-label="返回团队比赛"
        @back="goBack"
        ><template #actions
          ><el-button v-if="detail.can_submit" @click="openEventTicket"
            >比赛反馈</el-button
          ></template
        ></EventWorkspaceHeader
      >

      <main class="contest-content">
        <template v-if="section === 'description'">
          <div class="two-column">
            <section class="panel description-panel">
              <div class="section-heading">
                <h2>比赛说明</h2>
                <div class="contest-management-actions">
                  <el-button
                    v-if="detail.can_read_problems"
                    @click="attachmentsVisible = true"
                    >附件</el-button
                  ><el-button
                    v-if="detail.can_organize"
                    type="primary"
                    @click="
                      router.push(`/event-papers/contest/${contestID}/edit`)
                    "
                    >{{
                      detail.can_edit ? '组题' : '查看试卷与审核'
                    }}</el-button
                  ><el-button v-if="detail.can_edit" @click="editVisible = true"
                    >比赛设置</el-button
                  >
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
          <el-alert
            v-if="detail.lifecycle === 'closed'"
            :closable="false"
            type="info"
            title="本场已截止，现为只读状态。题目与附件仍可查看，等待管理员确认完结。"
          />
          <el-alert
            v-else-if="!detail.can_read_problems && !detail.can_organize"
            :closable="false"
            type="info"
            title="本场尚未开始或你不在参赛名单中，目前仅开放比赛说明。"
          />
        </template>

        <EventProblemList
          v-else-if="section === 'problems' && detail.can_read_problems"
          :items="problemItems"
          :base="`/contest/${contestID}`"
        >
          <template #actions
            ><el-button
              v-if="detail.can_edit"
              type="primary"
              @click="router.push(`/event-papers/contest/${contestID}/edit`)"
              >组题</el-button
            ></template
          >
        </EventProblemList>
        <EventProblemPanel
          v-else-if="section === 'problem' && selectedLink"
          :selected="{ ...selectedLink, score: 100 }"
          :items="problemItems"
          :records="problemRecords"
          :base="`/contest/${contestID}`"
          :asset-base="`/event-papers/contest/${contestID}/problems/${selectedLink.problem_id}/assets`"
          @refresh="loadRecords"
          @view-submission="viewSubmission"
        >
          <template #actions
            ><el-button v-if="detail.can_submit" @click="openProblemTicket"
              >工单修改</el-button
            ><el-button
              type="primary"
              :disabled="!detail.can_submit"
              @click="openSubmit"
              >提交代码</el-button
            ></template
          >
        </EventProblemPanel>

        <template
          v-else-if="
            section === 'scoreboard' &&
            detail.can_read_problems &&
            detail.contest.scoring_rule !== 'oi'
          "
        >
          <section class="contest-scoreboard">
            <LeaderboardBoard
              :data="scoreboardData"
              :loading="rankingLoading"
              :error="rankingError"
              empty-text="暂无比赛记录"
              :updated-at="rankingUpdatedAt"
              @refresh="loadRanking"
            />
          </section>
        </template>
        <section
          v-else-if="section === 'records' && canViewRecords"
          class="panel own-records"
        >
          <h2>本人提交</h2>
          <el-table
            :data="records.filter((r: any) => r.user_id === auth.user?.id)"
            ><el-table-column prop="id" label="提交编号" /><el-table-column
              prop="status"
              label="状态"
            /><el-table-column label="时间"
              ><template #default="{ row }">{{
                dateText(row.created_at)
              }}</template></el-table-column
            ><el-table-column label="操作"
              ><template #default="{ row }"
                ><el-button link @click="viewSubmission(row.id)"
                  >查看提交</el-button
                ></template
              ></el-table-column
            ></el-table
          ><el-button @click="loadRecords">刷新提交</el-button>
        </section>
        <el-result v-else title="此页面暂未开放"
          ><template #extra
            ><el-button
              @click="router.replace(`/contest/${contestID}/description`)"
              >返回比赛说明</el-button
            ></template
          ></el-result
        >
      </main>

      <el-dialog
        v-model="attachmentsVisible"
        title="比赛附件"
        width="min(760px, calc(100vw - 24px))"
        destroy-on-close
        ><EventAttachments
          v-if="detail.can_read_problems"
          kind="contest"
          :event-id="contestID"
          :editable="detail.can_edit"
          :revision="detail.contest.review_revision"
          @changed="loadDetail"
      /></el-dialog>
      <TicketDialog
        v-model="ticketVisible"
        :context="{
          kind: 'contest',
          id: contestID,
          problemID: ticketProblemID
        }"
      />
      <SubmissionDetailDialog
        v-model="submissionVisible"
        :detail="submissionDetail"
      />
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
      <el-dialog
        v-model="editVisible"
        title="比赛设置"
        width="min(620px, calc(100vw - 24px))"
        :before-close="closeSettings"
        :close-on-click-modal="false"
        ><el-form label-position="top" :disabled="editing"
          ><el-form-item label="比赛标题"
            ><el-input v-model="editForm.title" /></el-form-item
          ><el-form-item label="开始时间"
            ><el-date-picker
              v-model="editForm.starts_at"
              type="datetime"
              value-format="YYYY-MM-DDTHH:mm:ssZ" /></el-form-item
          ><el-form-item label="时长（分钟）"
            ><el-input-number
              v-model="editForm.duration_minutes"
              :min="15" /></el-form-item
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
          ><el-form-item label="奖项设置"
            ><TeamContestAwardFields
              v-if="editVisible"
              v-model:gold-award-percent="editForm.gold_award_percent"
              v-model:silver-award-percent="editForm.silver_award_percent"
              v-model:bronze-award-percent="editForm.bronze_award_percent"
              @validity-change="awardInputValid = $event"
          /></el-form-item>
          <el-form-item label="说明"
            ><el-input
              v-model="editForm.description"
              type="textarea"
              :autosize="{ minRows: 6 }"
              resize="vertical"
              aria-label="比赛说明"
              maxlength="20000" /></el-form-item></el-form
        ><template #footer
          ><el-button :disabled="editing" @click="closeSettings()"
            >取消</el-button
          ><el-button
            type="primary"
            :loading="editing"
            :disabled="!awardInputValid"
            @click="saveContest"
            >保存</el-button
          ></template
        ></el-dialog
      >
    </div>
    <el-skeleton v-else :rows="12" animated class="loading" />
  </section>
</template>

<script setup lang="ts">
import { eventLifecycle, lifecycleLabels } from '../../api/modules/events'
import { useSubmissionStreams } from '../../features/submissions/live'
import { apiErrorMessage } from '../../api/errors'
import EventAttachments from '../../components/events/EventAttachments.vue'
import TicketDialog from '../../components/tickets/TicketDialog.vue'

import { submissionLanguage } from '../../features/submissions/presentation'
import { ElMessage, ElMessageBox } from 'element-plus'
import TeamContestAwardFields from '../../components/TeamContestAwardFields.vue'
import { contestAwardValidationError } from '../../features/teams/contestAwards'
import { useEditSnapshot } from '../../features/events/editSnapshot'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import {
  onBeforeRouteLeave,
  onBeforeRouteUpdate,
  useRoute,
  useRouter
} from 'vue-router'
import {
  generatedApi,
  getLatestSubmissions,
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
const submissionStreams = useSubmissionStreams()
const route = useRoute(),
  router = useRouter(),
  auth = useAuthStore()
const detail = ref<any>(null),
  ranking = ref<any>({ rows: [], problems: [] }),
  records = ref<Submission[]>([]),
  rankingLoading = ref(false),
  rankingError = ref(''),
  rankingLoadedAt = ref<Date | null>(null),
  submitVisible = ref(false),
  submitting = ref(false),
  language = ref('cpp'),
  source = ref(''),
  liveStatus = ref<any>(null),
  composer = ref<any>(null),
  editVisible = ref(false),
  editing = ref(false)
let stream: AuthenticatedEventSource | null = null
let rankingDisposed = false
let detailTimer: number | undefined, deadlineTimer: number | undefined
let detailLoading = false
const ticketVisible = ref(false),
  ticketProblemID = ref<number | undefined>()
function openEventTicket() {
  ticketProblemID.value = undefined
  ticketVisible.value = true
}
function openProblemTicket() {
  if (selectedLink.value) {
    ticketProblemID.value = selectedLink.value.problem_id
    ticketVisible.value = true
  }
}
const scoringOptions = [
  { label: 'OI', value: 'oi' },
  { label: 'IOI', value: 'ioi' },
  { label: 'ACM', value: 'acm' }
]
const settingsDraft = useEditSnapshot({
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
const editForm = settingsDraft.form
const awardInputValid = ref(true)
const contestID = computed(() => Number(route.params.contestId))
const section = computed(() =>
  route.name === 'contest-description'
    ? 'description'
    : route.name === 'contest-problems'
      ? 'problems'
      : route.name === 'contest-problem'
        ? 'problem'
        : route.name === 'contest-records'
          ? 'records'
          : 'scoreboard'
)
const selectedLink = computed(
  () =>
    detail.value?.problems?.find(
      (i: any) =>
        String(i.label).toLowerCase() ===
        String(route.params.label || '').toLowerCase()
    ) || null
)
const problemItems = computed(() =>
  (detail.value?.problems || []).map((link: any) => ({ ...link, score: 100 }))
)
const submissionVisible = ref(false),
  submissionDetail = ref<any>(null)
async function viewSubmission(id: number) {
  try {
    submissionDetail.value = (
      await generatedApi.getSubmission<any>({ path: { id: id } })
    ).data
    submissionVisible.value = true
  } catch (err: any) {
    ElMessage.error(apiErrorMessage(err, '提交加载失败'))
  }
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
const attachmentsVisible = ref(false)
const canViewRecords = computed(() => Boolean(detail.value?.can_read_problems))
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
  if (detailLoading) return
  detailLoading = true
  try {
    detail.value = (
      await generatedApi.getTeamContestViaContestsContestId<any>({
        path: { contest_id: contestID.value }
      })
    ).data
    if (editVisible.value)
      settingsDraft.sync(
        detail.value.contest,
        detail.value.contest.review_revision
      )
    if (deadlineTimer) window.clearTimeout(deadlineTimer)
    if (detail.value.lifecycle === 'running' && detail.value.contest.ends_at) {
      const delay =
        Date.parse(detail.value.contest.ends_at) - Date.parse(detail.value.now)
      deadlineTimer = window.setTimeout(
        () => {
          if (detail.value) {
            detail.value.can_submit = false
            submitVisible.value = false
          }
          void loadDetail()
        },
        Math.min(Math.max(100, delay + 100), 2147483647)
      )
    }
    if (!detail.value.can_read_problems) {
      ranking.value = { rows: [], problems: [] }
      submitVisible.value = false
      if (
        section.value !== 'description' &&
        !(section.value === 'records' && canViewRecords.value)
      )
        await router.replace(`/contest/${contestID.value}/description`)
    }
    if (
      detail.value.contest.scoring_rule === 'oi' &&
      section.value === 'scoreboard'
    )
      await router.replace(`/contest/${contestID.value}/problems`)
    if (route.query.manage === 'edit' && detail.value.can_edit) {
      editVisible.value = true
      await router.replace({ path: route.path, query: {} })
    }
    await loadRecords()
  } catch (e: any) {
    if (e.response?.data?.code === 'EVENT_COMPLETED') {
      await router.replace(`/competitions/contest/${contestID.value}`)
      return
    }
    ElMessage.error(apiErrorMessage(e))
    // A transient background refresh failure must not unmount an edited form.
    if (!detail.value || [401, 403, 404].includes(e.response?.status)) {
      if (editVisible.value && settingsDraft.dirty.value) return
      detail.value = null
      attachmentsVisible.value = false
      submitVisible.value = false
      await router.replace('/teams')
    }
  } finally {
    detailLoading = false
  }
}
async function loadRecords() {
  if (!contestID.value || !canViewRecords.value) {
    records.value = []
    return
  }
  try {
    records.value =
      (
        await generatedApi.listTeamContestSubmissionsViaContestsContestIdSubmissions<any>(
          { path: { contest_id: contestID.value } }
        )
      ).data || []
  } catch {
    records.value = []
  }
}
async function loadRanking() {
  if (rankingDisposed || !detail.value?.can_read_problems) return
  if (detail.value?.contest.scoring_rule === 'oi' || rankingLoading.value)
    return
  rankingLoading.value = true
  rankingError.value = ''
  try {
    ranking.value = (
      await generatedApi.teamContestRankingViaContestsContestIdRanking<any>({
        path: { contest_id: contestID.value }
      })
    ).data || {
      rows: [],
      problems: []
    }
    rankingLoadedAt.value = new Date()
  } catch (e: any) {
    ranking.value = { rows: [], problems: [] }
    rankingError.value = apiErrorMessage(e)
  } finally {
    rankingLoading.value = false
  }
}
function goBack() {
  void router.push(
    detail.value?.team?.slug
      ? `/teams/${detail.value.team.slug}/contests`
      : '/teams'
  )
}
function dateText(v?: string) {
  return v ? formatDateTime(v) : '待定'
}
async function openSubmit() {
  if (!selectedLink.value) return
  const latest = (
    await getLatestSubmissions(
      { team_contest_id: contestID.value },
      selectedLink.value.problem_id
    )
  )[0]
  language.value = latest?.language || 'cpp'
  source.value =
    loadSubmissionDraft(draftContext.value, language.value) ??
    latest?.source_code ??
    ''
  liveStatus.value = latest || null
  submitVisible.value = true
}
async function submitSolution() {
  if (!source.value.trim()) return ElMessage.warning('请输入代码')
  submitting.value = true
  try {
    const { data } =
      await generatedApi.createTeamContestSubmissionViaContestsContestIdSubmissions<any>(
        {
          path: { contest_id: contestID.value },
          body: {
            problem_id: selectedLink.value.problem_id,
            language: submissionLanguage(language.value),
            source_code: source.value
          }
        }
      )
    composer.value?.clearDraft()
    liveStatus.value = data
    stream?.close()
    stream = submissionStreams.watch(
      data.id,
      (state) => {
        liveStatus.value = state
      },
      async () => {
        await Promise.all([loadDetail(), loadRanking()])
      }
    )
    ElMessage.success('代码已提交')
  } catch (e: any) {
    ElMessage.error(apiErrorMessage(e))
  } finally {
    submitting.value = false
  }
}
async function saveContest() {
  const awardError = !awardInputValid.value
    ? '请检查奖项设置和换算基数'
    : contestAwardValidationError(editForm)
  if (awardError) {
    ElMessage.warning(awardError)
    return
  }
  if (editForm.scoring_rule === 'oi') editForm.freeze_enabled = false
  editing.value = true
  try {
    await generatedApi.updateTeamContestViaContestsContestId<any>({
      path: { contest_id: contestID.value },
      body: { ...editForm, revision: settingsDraft.revision.value }
    })
    settingsDraft.markSaved()
    editVisible.value = false
    await loadDetail()
    ElMessage.success('设置已保存')
  } catch (e: any) {
    ElMessage.error(apiErrorMessage(e))
  } finally {
    editing.value = false
  }
}
watch(
  editVisible,
  (open) => {
    if (open && detail.value)
      settingsDraft.begin(
        detail.value.contest,
        detail.value.contest.review_revision
      )
  },
  { flush: 'sync' }
)
async function confirmSettingsLeave() {
  if (!editVisible.value || !settingsDraft.dirty.value) return true
  try {
    await ElMessageBox.confirm(
      '比赛设置有尚未保存的修改，离开将丢弃这些修改。',
      '未保存的比赛设置',
      { confirmButtonText: '丢弃并离开', cancelButtonText: '继续编辑' }
    )
    return true
  } catch {
    return false
  }
}
async function closeSettings(done?: () => void) {
  if (editing.value || !(await confirmSettingsLeave())) return
  if (done) done()
  else editVisible.value = false
}
function protectSettings(event: BeforeUnloadEvent) {
  if (editVisible.value && settingsDraft.dirty.value) {
    event.preventDefault()
    event.returnValue = ''
  }
}
onBeforeRouteLeave(confirmSettingsLeave)
onBeforeRouteUpdate(async (to, from) => {
  if (to.path === from.path) return
  if (!(await confirmSettingsLeave())) return false
  editVisible.value = false
})
onMounted(() => window.addEventListener('beforeunload', protectSettings))
onBeforeUnmount(() =>
  window.removeEventListener('beforeunload', protectSettings)
)
watch(
  () => route.fullPath,
  async () => {
    if (
      !detail.value ||
      Number(route.params.contestId) !== detail.value.contest.id
    )
      await loadDetail()
    if (section.value === 'scoreboard') await loadRanking()
  }
)
onMounted(async () => {
  await loadDetail()
  detailTimer = window.setInterval(() => void loadDetail(), 15000)
  if (section.value === 'scoreboard') await loadRanking()
})
onBeforeUnmount(() => {
  rankingDisposed = true
  if (detailTimer) window.clearInterval(detailTimer)
  if (deadlineTimer) window.clearTimeout(deadlineTimer)
  stream?.close()
})
</script>

<style scoped>
.contest-page {
  min-height: 100%;
  padding: 24px 20px 58px;
  background: color-mix(in srgb, var(--surface) 74%, var(--color-canvas));
}
.contest-shell {
  width: min(1320px, 100%);
  margin: 0 auto;
}
.contest-content {
  padding-top: 16px;
}
.contest-scoreboard {
  min-width: 0;
  overflow: hidden;
}
.two-column {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 16px;
}
.description-panel,
.meta-panel {
  padding: 20px;
}
.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 18px;
}
.contest-management-actions {
  justify-content: flex-start;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.contest-management-actions .el-button {
  margin-left: 0;
}
.section-heading h2 {
  margin: 0;
}
.description-copy {
  min-height: 240px;
  margin-top: 18px;
  padding-top: 18px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  line-height: 1.8;
  border-top: 1px solid var(--border);
}
.meta-panel h3 {
  margin-top: 0;
}
.meta-panel dl div {
  display: grid;
  grid-template-columns: 88px 1fr;
  gap: 8px;
  padding: 12px 0;
  border-top: 1px solid var(--border);
}
.meta-panel dt {
  color: var(--muted);
}
.meta-panel dd {
  margin: 0;
  text-align: right;
}
.loading {
  max-width: 1320px;
  margin: 0 auto;
}
@media (max-width: 880px) {
  .two-column {
    grid-template-columns: minmax(0, 1fr);
  }
}
@media (max-width: 600px) {
  .contest-page {
    padding: 16px 10px 36px;
  }
}
</style>
