<template>
  <section class="ticket-page">
    <header class="ticket-header">
      <div>
        <span class="eyebrow">CHANGE REQUEST</span>
        <h1>工单修改</h1>
        <p>题目新增、替换、删除与测试点变更统一由管理员执行。</p>
      </div>
      <el-radio-group v-model="mode" @change="switchMode">
        <el-radio-button value="mine">我的工单</el-radio-button>
        <el-radio-button value="new">发起工单</el-radio-button>
      </el-radio-group>
    </header>

    <section v-if="mode === 'mine'" class="panel ticket-list" v-loading="loading">
      <div class="section-heading">
        <div>
          <h2>处理进度</h2>
          <p>管理员将根据说明重新上传完整题包，执行后会生成不可变版本。</p>
        </div>
        <el-button @click="loadTickets">刷新</el-button>
      </div>
      <el-table :data="tickets" row-key="id">
        <el-table-column label="工单" width="92"
          ><template #default="{ row }">#{{ row.id }}</template></el-table-column
        >
        <el-table-column label="类型" width="90"
          ><template #default="{ row }"
            ><el-tag :type="actionType(row.action)">{{ actionLabel(row.action) }}</el-tag></template
          ></el-table-column
        >
        <el-table-column label="目标题目" min-width="180"
          ><template #default="{ row }">{{
            row.problem
              ? `${row.problem.display_code || `#${row.problem.id}`} · ${row.problem.title}`
              : scopeLabel(row.target_scope)
          }}</template></el-table-column
        >
        <el-table-column
          prop="description"
          label="修改说明"
          min-width="260"
          show-overflow-tooltip
        />
        <el-table-column label="状态" width="110"
          ><template #default="{ row }"
            ><el-tag :type="statusType(row.status)" effect="plain">{{
              statusLabel(row.status)
            }}</el-tag></template
          ></el-table-column
        >
        <el-table-column label="提交时间" width="170"
          ><template #default="{ row }">{{
            formatDateTime(row.created_at)
          }}</template></el-table-column
        >
        <el-table-column label="操作" width="100" align="right"
          ><template #default="{ row }"
            ><el-button v-if="row.status === 'pending'" type="danger" link @click="cancel(row)"
              >取消</el-button
            ></template
          ></el-table-column
        >
      </el-table>
      <el-empty v-if="!loading && !tickets.length" description="还没有工单" />
    </section>

    <section v-else class="new-ticket-layout">
      <main class="panel ticket-form-panel">
        <h2>描述修改需求</h2>
        <el-alert
          title="附件必须包含修改后的正确内容"
          description="所有工单均须上传参考附件。替换与删除采用覆盖性处理；管理员审核后仍会上传或确认最终版本。"
          type="info"
          :closable="false"
          show-icon
        />
        <el-form label-position="top" class="ticket-form">
          <el-form-item label="工单类型" required
            ><el-segmented v-model="form.action" :options="actionOptions"
          /></el-form-item>
          <el-form-item v-if="form.action !== 'create'" label="目标题目" required>
            <el-select
              v-model="form.problem_id"
              filterable
              placeholder="选择题目"
              style="width: 100%"
              ><el-option
                v-for="problem in ownedProblems"
                :key="problem.id"
                :label="`${problem.display_code || `#${problem.id}`} · ${problem.title}`"
                :value="problem.id"
            /></el-select>
          </el-form-item>
          <el-form-item v-if="form.action === 'replace'" label="执行后范围" required>
            <el-radio-group v-model="form.target_scope">
              <el-radio value="public">公共题库（预备题将发布）</el-radio>
              <el-radio value="prepared">保留在预备题库</el-radio>
            </el-radio-group>
          </el-form-item>
          <template v-if="form.action === 'create'">
            <el-form-item label="新增范围" required
              ><el-radio-group v-model="form.target_scope"
                ><el-radio value="public">公共题库</el-radio
                ><el-radio value="prepared">预备题库</el-radio
                ><el-radio value="team_problem_set">指定团队题单</el-radio></el-radio-group
              ></el-form-item
            >
            <el-form-item
              v-if="form.target_scope === 'team_problem_set'"
              label="团队题单 ID"
              required
              ><el-input-number
                v-model="form.team_problem_set_id"
                :min="1"
                controls-position="right"
            /></el-form-item>
          </template>
          <el-form-item v-if="form.action !== 'create'" label="处理方式" required>
            <el-checkbox v-model="overwriteConfirmed">
              {{ form.action === 'replace' ? '覆盖当前题目并生成新版本' : '覆盖归档当前题目' }}
            </el-checkbox>
          </el-form-item>
          <el-form-item label="修改说明" required
            ><el-input
              v-model="form.description"
              type="textarea"
              :rows="7"
              maxlength="2000"
              show-word-limit
              placeholder="说明问题、预期结果、受影响测试点及必要的兼容要求（至少 8 个字）"
          /></el-form-item>
          <el-form-item label="参考附件" required
            ><el-upload
              :key="uploadKey"
              :auto-upload="false"
              :limit="1"
              :on-change="pickAttachment"
              :on-remove="clearAttachment"
              ><el-button type="primary" plain>上传修改后的正确内容</el-button
              ><template #tip
                ><small class="upload-tip"
                  >必填。请上传完整题包、修正题面或删除依据，最大 32
                  MB；管理员将以此为审核基准。</small
                ></template
              ></el-upload
            ></el-form-item
          >
          <div class="form-actions">
            <el-button @click="router.push('/problems')">返回题库</el-button
            ><el-button type="primary" :loading="submitting" @click="submit">提交工单</el-button>
          </div>
        </el-form>
      </main>
      <aside class="panel process-card">
        <h3>处理流程</h3>
        <ol>
          <li><strong>提交正确内容</strong><span>说明需求并附上修改后的文件。</span></li>
          <li><strong>管理员核对</strong><span>比对题目版本、附件与影响场次。</span></li>
          <li><strong>覆盖性执行</strong><span>替换或删除不做局部修改，只覆盖当前内容。</span></li>
          <li>
            <strong>固定历史版本</strong><span>未开始场次自动更新，已开始场次保持原版本。</span>
          </li>
        </ol>
      </aside>
    </section>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  client,
  type Problem,
  type ProblemChangeAction,
  type ProblemChangeStatus,
  type ProblemChangeTicket
} from '../../api/client'
import { validateProblemChangeTicketDraft } from '../../features/problems/changeTicket'
import { formatDateTime } from '../../features/time'
import { useAuthStore } from '../../stores/auth'

const route = useRoute(),
  router = useRouter(),
  auth = useAuthStore()
const mode = ref(route.path.endsWith('/new') ? 'new' : 'mine')
const loading = ref(false),
  submitting = ref(false)
const tickets = ref<ProblemChangeTicket[]>([]),
  problems = ref<Problem[]>([])
const attachment = ref<File | null>(null)
const overwriteConfirmed = ref(false)
const uploadKey = ref(0)
const actionOptions = [
  { label: '新增', value: 'create' },
  { label: '替换', value: 'replace' },
  { label: '删除', value: 'archive' }
]
const form = reactive({
  action: (route.query.action as ProblemChangeAction) || 'create',
  problem_id: Number(route.query.problem_id) || (undefined as number | undefined),
  target_scope: String(route.query.target_scope || 'public'),
  team_problem_set_id: Number(route.query.team_problem_set_id) || (undefined as number | undefined),
  description: ''
})
const ownedProblems = computed(() =>
  auth.role === 'admin'
    ? problems.value
    : problems.value.filter((item) => item.owner_id === auth.user?.id)
)

async function loadTickets() {
  loading.value = true
  try {
    tickets.value =
      (await client.get<ProblemChangeTicket[]>('/problem-change-tickets/mine')).data || []
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    loading.value = false
  }
}
async function loadProblems() {
  try {
    problems.value =
      (await client.get<Problem[]>('/problem-change-tickets/eligible-problems')).data || []
  } catch {
    problems.value = []
  }
}
function switchMode(value: string | number | boolean) {
  void router.push(value === 'new' ? '/problem-changes/new' : '/problem-changes')
}
function pickAttachment(file: { raw?: File }) {
  attachment.value = file.raw || null
}
function clearAttachment() {
  attachment.value = null
}
async function submit() {
  const validationMessage = validateProblemChangeTicketDraft({
    action: form.action,
    problemID: form.problem_id,
    targetScope: form.target_scope,
    teamProblemSetID: form.team_problem_set_id,
    description: form.description,
    attachment: attachment.value,
    overwriteConfirmed: overwriteConfirmed.value
  })
  if (validationMessage) return ElMessage.warning(validationMessage)
  const data = new FormData()
  data.append('action', form.action)
  data.append('description', form.description.trim())
  data.append('target_scope', form.action === 'archive' ? 'public' : form.target_scope)
  if (form.problem_id) data.append('problem_id', String(form.problem_id))
  if (form.team_problem_set_id) data.append('team_problem_set_id', String(form.team_problem_set_id))
  if (attachment.value) data.append('attachment', attachment.value)
  submitting.value = true
  try {
    await client.post('/problem-change-tickets', data)
    ElMessage.success('工单已提交')
    form.description = ''
    attachment.value = null
    overwriteConfirmed.value = false
    uploadKey.value += 1
    await router.push('/problem-changes')
    await loadTickets()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    submitting.value = false
  }
}
async function cancel(ticket: ProblemChangeTicket) {
  try {
    await ElMessageBox.confirm(`确认取消工单 #${ticket.id}？`, '取消工单')
    await client.post(`/problem-change-tickets/${ticket.id}/cancel`)
    await loadTickets()
  } catch (e: any) {
    if (e !== 'cancel' && e !== 'close') ElMessage.error(e.response?.data?.error || e.message)
  }
}
function actionLabel(v: ProblemChangeAction) {
  return v === 'create' ? '新增' : v === 'replace' ? '替换' : '删除'
}
function actionType(v: ProblemChangeAction): 'success' | 'warning' | 'danger' {
  return v === 'create' ? 'success' : v === 'replace' ? 'warning' : 'danger'
}
function statusLabel(v: ProblemChangeStatus) {
  return (
    {
      pending: '待处理',
      processing: '处理中',
      completed: '已完成',
      rejected: '已驳回',
      cancelled: '已取消'
    } as const
  )[v]
}
function statusType(v: ProblemChangeStatus): 'success' | 'warning' | 'danger' | 'info' {
  return v === 'completed'
    ? 'success'
    : v === 'pending' || v === 'processing'
      ? 'warning'
      : v === 'rejected'
        ? 'danger'
        : 'info'
}
function scopeLabel(v: string) {
  return v === 'prepared'
    ? '新增至预备题库'
    : v === 'team_problem_set'
      ? '新增至团队题单'
      : '新增至公共题库'
}
watch(
  () => form.action,
  () => {
    overwriteConfirmed.value = false
  }
)
watch(
  () => route.path,
  (path) => {
    mode.value = path.endsWith('/new') ? 'new' : 'mine'
  }
)
onMounted(async () => {
  await Promise.all([loadProblems(), loadTickets()])
})
</script>

<style scoped>
.ticket-page {
  width: min(1180px, calc(100% - 32px));
  margin: 0 auto;
  padding: 30px 0 56px;
}
.ticket-header,
.section-heading,
.form-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
}
.ticket-header {
  margin-bottom: 20px;
}
.ticket-header h1 {
  margin: 6px 0 4px;
}
.ticket-header p,
.section-heading p {
  margin: 0;
  color: var(--muted);
}
.eyebrow {
  color: var(--accent);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.16em;
}
.ticket-list,
.ticket-form-panel,
.process-card {
  padding: 20px;
}
.section-heading {
  margin-bottom: 16px;
}
.section-heading h2 {
  margin: 0 0 5px;
}
.new-ticket-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 300px;
  gap: 16px;
}
.ticket-form-panel h2 {
  margin-top: 0;
}
.ticket-form {
  margin-top: 20px;
}
.upload-tip {
  display: block;
  margin-top: 7px;
  color: var(--muted);
}
.form-actions {
  justify-content: flex-end;
  padding-top: 8px;
}
.process-card {
  align-self: start;
}
.process-card h3 {
  margin-top: 0;
}
.process-card ol {
  display: grid;
  gap: 18px;
  margin: 0;
  padding-left: 22px;
}
.process-card li {
  padding-left: 4px;
}
.process-card li::marker {
  color: var(--accent);
  font-weight: 800;
}
.process-card strong,
.process-card span {
  display: block;
}
.process-card span {
  margin-top: 3px;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.6;
}
@media (max-width: 760px) {
  .ticket-header,
  .section-heading {
    align-items: stretch;
    flex-direction: column;
  }
  .new-ticket-layout {
    grid-template-columns: 1fr;
  }
  .process-card {
    order: -1;
  }
  .ticket-page {
    width: min(100% - 24px, 1180px);
  }
}
</style>
