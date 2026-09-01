<template>
  <section class="admin-ticket-page">
    <div class="page-heading">
      <div>
        <span>PROBLEM OPERATIONS</span>
        <h2>工单管理</h2>
        <p>这里只处理前台题目数据工单；出题资格已移至“用户与权限”。</p>
      </div>
      <el-button :loading="loading" @click="load">刷新</el-button>
    </div>

    <section class="summary-grid" aria-label="工单概况">
      <article>
        <span>待处理</span><strong>{{ count('pending') }}</strong>
      </article>
      <article>
        <span>处理中</span><strong>{{ count('processing') }}</strong>
      </article>
      <article>
        <span>已完成</span><strong>{{ count('completed') }}</strong>
      </article>
    </section>

    <section class="panel ticket-panel">
      <div class="filters">
        <el-select v-model="statusFilter" aria-label="筛选工单状态">
          <el-option label="全部状态" value="" />
          <el-option label="待处理" value="pending" />
          <el-option label="处理中" value="processing" />
          <el-option label="已完成" value="completed" />
          <el-option label="已驳回" value="rejected" />
          <el-option label="已取消" value="cancelled" />
        </el-select>
      </div>
      <el-table v-loading="loading" :data="filtered" row-key="id">
        <el-table-column label="工单" width="85"
          ><template #default="{ row }">#{{ row.id }}</template></el-table-column
        >
        <el-table-column label="类型" width="110"
          ><template #default="{ row }"
            ><el-tag :type="actionType(row.action)" effect="plain">{{
              actionLabel(row.action)
            }}</el-tag></template
          ></el-table-column
        >
        <el-table-column label="申请人" width="140"
          ><template #default="{ row }">{{
            row.requester?.name || `#${row.requester_id}`
          }}</template></el-table-column
        >
        <el-table-column label="目标" min-width="190"
          ><template #default="{ row }">{{
            row.problem
              ? `${row.problem.display_code || row.problem.id} · ${row.problem.title}`
              : '新题目'
          }}</template></el-table-column
        >
        <el-table-column
          prop="description"
          label="修改说明"
          min-width="250"
          show-overflow-tooltip
        />
        <el-table-column label="正确内容" width="130"
          ><template #default="{ row }"
            ><span :class="row.attachment_name ? 'attachment-ready' : 'attachment-missing'">{{
              row.attachment_name ? '已提交' : '历史工单缺失'
            }}</span></template
          ></el-table-column
        >
        <el-table-column label="状态" width="100"
          ><template #default="{ row }"
            ><el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template
          ></el-table-column
        >
        <el-table-column label="提交时间" width="170"
          ><template #default="{ row }">{{
            formatDateTime(row.created_at)
          }}</template></el-table-column
        >
        <el-table-column label="操作" width="100" fixed="right"
          ><template #default="{ row }"
            ><el-button link type="primary" @click="open(row)">查看</el-button></template
          ></el-table-column
        >
      </el-table>
    </section>

    <el-drawer v-model="drawer" title="工单处理" size="min(590px, 100%)">
      <template v-if="active">
        <div class="ticket-detail">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="工单"
              >#{{ active.id }} · {{ actionLabel(active.action) }}</el-descriptions-item
            >
            <el-descriptions-item label="申请人"
              >{{ active.requester?.name }}（{{ active.requester?.email }}）</el-descriptions-item
            >
            <el-descriptions-item label="目标题目">{{
              active.problem?.title || '新建题目'
            }}</el-descriptions-item>
            <el-descriptions-item label="修改说明">{{ active.description }}</el-descriptions-item>
            <el-descriptions-item label="修改后内容">
              <el-button
                v-if="active.attachment_name"
                link
                type="primary"
                @click="downloadAttachment(active)"
                >{{ active.attachment_name }}</el-button
              >
              <el-tag v-else type="danger" effect="plain">历史工单未附文件</el-tag>
            </el-descriptions-item>
          </el-descriptions>

          <section v-if="active.impact_summary" class="impact-card">
            <h3>影响范围</h3>
            <div>
              <span
                >未开始考试<strong>{{ active.impact_summary.future_exams }}</strong></span
              >
              <span
                >已固定考试<strong>{{ active.impact_summary.pinned_exams }}</strong></span
              >
              <span
                >未开始比赛<strong>{{ active.impact_summary.future_contests }}</strong></span
              >
              <span
                >已固定比赛<strong>{{ active.impact_summary.pinned_contests }}</strong></span
              >
              <span
                >历史提交<strong>{{ active.impact_summary.historical_submissions }}</strong></span
              >
            </div>
            <p>覆盖替换只更新公共题目与未开始场次；已开始场次和历史提交保持原版本。</p>
          </section>

          <template v-if="active.status === 'pending'">
            <el-form label-position="top">
              <el-form-item v-if="active.action !== 'create'" label="处理方式" required>
                <el-radio-group v-model="applyForm.operation_mode">
                  <el-radio value="overwrite">{{
                    active.action === 'replace' ? '覆盖当前题目并生成新版本' : '覆盖归档当前题目'
                  }}</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item v-if="active.action !== 'archive'" label="管理员最终完整题包" required>
                <el-upload
                  :key="packageUploadKey"
                  :auto-upload="false"
                  :limit="1"
                  accept=".zip"
                  :on-change="pickPackage"
                  :on-remove="clearPackage"
                >
                  <el-button>选择 ZIP 题包</el-button>
                  <template #tip><small>请基于申请人附件复核后上传最终完整版本。</small></template>
                </el-upload>
              </el-form-item>
              <el-form-item v-if="active.action !== 'archive'" label="难度">
                <el-select v-model="applyForm.difficulty" clearable
                  ><el-option
                    v-for="value in difficulties"
                    :key="value"
                    :label="value"
                    :value="value"
                /></el-select>
              </el-form-item>
              <el-form-item label="处理说明"
                ><el-input v-model="applyForm.note" type="textarea" :rows="4"
              /></el-form-item>
            </el-form>
            <div class="drawer-actions">
              <el-button type="danger" plain @click="reject">驳回</el-button>
              <el-button type="primary" :loading="applying" @click="apply">{{
                applyButtonLabel
              }}</el-button>
            </div>
          </template>
          <el-alert
            v-else
            :title="`工单${statusLabel(active.status)}`"
            :description="active.resolution_note || '暂无处理说明'"
            :type="active.status === 'completed' ? 'success' : 'info'"
            :closable="false"
          />
        </div>
      </template>
    </el-drawer>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import {
  client,
  type ProblemChangeAction,
  type ProblemChangeStatus,
  type ProblemChangeTicket
} from '../../api/client'
import { problemChangeOperationMode } from '../../features/problems/changeTicket'
import { problemDifficultyOptions } from '../../features/problems/problemMeta'
import { formatDateTime } from '../../features/time'

const loading = ref(false)
const applying = ref(false)
const drawer = ref(false)
const tickets = ref<ProblemChangeTicket[]>([])
const active = ref<ProblemChangeTicket | null>(null)
const statusFilter = ref('pending')
const packageFile = ref<File | null>(null)
const packageUploadKey = ref(0)
const difficulties = problemDifficultyOptions
const applyForm = reactive({
  difficulty: '',
  note: '',
  operation_mode: 'overwrite'
})
const filtered = computed(() =>
  statusFilter.value
    ? tickets.value.filter((item) => item.status === statusFilter.value)
    : tickets.value
)
const applyButtonLabel = computed(() =>
  active.value?.action === 'replace'
    ? '覆盖并生成新版本'
    : active.value?.action === 'archive'
      ? '覆盖归档'
      : '上传并创建'
)

async function load() {
  loading.value = true
  try {
    tickets.value = (await client.get<ProblemChangeTicket[]>('/problem-change-tickets')).data || []
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || error.message)
  } finally {
    loading.value = false
  }
}

async function open(row: ProblemChangeTicket) {
  active.value = row
  packageFile.value = null
  packageUploadKey.value += 1
  Object.assign(applyForm, {
    difficulty: row.problem?.difficulty || '',
    note: '',
    operation_mode: 'overwrite'
  })
  drawer.value = true
  try {
    active.value = (await client.get<ProblemChangeTicket>(`/problem-change-tickets/${row.id}`)).data
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || error.message)
  }
}

function pickPackage(file: { raw?: File }) {
  packageFile.value = file.raw || null
}
function clearPackage() {
  packageFile.value = null
}

async function apply() {
  if (!active.value) return
  if (!active.value.attachment_name)
    return ElMessage.warning('该历史工单缺少申请人附件，请先驳回并要求重新提交')
  if (active.value.action !== 'archive' && !packageFile.value)
    return ElMessage.warning('请上传完整最终题包')
  if (active.value.action !== 'create' && applyForm.operation_mode !== 'overwrite')
    return ElMessage.warning('请选择覆盖性操作')
  if (active.value.action !== 'create') {
    try {
      await ElMessageBox.confirm(
        active.value.action === 'replace'
          ? '确认用管理员最终题包覆盖当前题目版本？'
          : '确认覆盖归档当前题目？历史场次与提交仍会保留。',
        '确认覆盖操作',
        {
          type: 'warning',
          confirmButtonText: '确认覆盖',
          cancelButtonText: '取消'
        }
      )
    } catch (error) {
      if (error === 'cancel' || error === 'close') return
      throw error
    }
  }
  applying.value = true
  try {
    const data = new FormData()
    if (packageFile.value) data.append('package', packageFile.value)
    data.append('difficulty', applyForm.difficulty)
    data.append('resolution_note', applyForm.note.trim())
    data.append('operation_mode', problemChangeOperationMode(active.value.action))
    await client.post(`/problem-change-tickets/${active.value.id}/apply`, data)
    ElMessage.success(
      active.value.action === 'archive' ? '题目已覆盖归档' : '工单已执行并生成题目版本'
    )
    drawer.value = false
    await load()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || error.message)
  } finally {
    applying.value = false
  }
}

async function reject() {
  if (!active.value) return
  try {
    const note = await ElMessageBox.prompt('填写驳回原因', '驳回工单', {
      inputType: 'textarea',
      inputValidator: (value) => Boolean(value.trim()) || '必须填写原因'
    })
    await client.post(`/problem-change-tickets/${active.value.id}/reject`, {
      note: note.value.trim()
    })
    drawer.value = false
    await load()
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close')
      ElMessage.error(error.response?.data?.error || error.message)
  }
}

async function downloadAttachment(ticket: ProblemChangeTicket) {
  try {
    const response = await client.get(`/problem-change-tickets/${ticket.id}/attachment`, {
      responseType: 'blob'
    })
    const url = URL.createObjectURL(response.data)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = ticket.attachment_name || `ticket-${ticket.id}-attachment`
    anchor.click()
    URL.revokeObjectURL(url)
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || error.message)
  }
}

function count(status: ProblemChangeStatus) {
  return tickets.value.filter((item) => item.status === status).length
}
function actionLabel(value: ProblemChangeAction) {
  return value === 'create' ? '新增' : value === 'replace' ? '覆盖修改' : '覆盖删除'
}
function actionType(value: ProblemChangeAction): 'success' | 'warning' | 'danger' {
  return value === 'create' ? 'success' : value === 'replace' ? 'warning' : 'danger'
}
function statusLabel(value: ProblemChangeStatus) {
  return (
    {
      pending: '待处理',
      processing: '处理中',
      completed: '已完成',
      rejected: '已驳回',
      cancelled: '已取消'
    } as const
  )[value]
}
function statusType(value: ProblemChangeStatus): 'success' | 'warning' | 'danger' | 'info' {
  return value === 'completed'
    ? 'success'
    : value === 'pending' || value === 'processing'
      ? 'warning'
      : value === 'rejected'
        ? 'danger'
        : 'info'
}

onMounted(load)
</script>

<style scoped>
.admin-ticket-page {
  padding: 28px;
}
.page-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}
.page-heading span {
  color: #135ecb;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.14em;
}
.page-heading h2 {
  margin: 5px 0;
}
.page-heading p {
  margin: 0;
  color: var(--muted);
}
.summary-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-bottom: 14px;
}
.summary-grid article {
  padding: 18px;
  border: 1px solid var(--border);
  border-radius: 7px;
  background: var(--surface-strong);
}
.summary-grid span,
.summary-grid strong {
  display: block;
}
.summary-grid span {
  color: var(--muted);
  font-size: 13px;
}
.summary-grid strong {
  margin-top: 8px;
  font-size: 28px;
}
.ticket-panel {
  overflow: hidden;
}
.filters {
  width: 210px;
  padding: 14px 16px 0;
}
.ticket-detail {
  display: grid;
  gap: 18px;
}
.impact-card {
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 8px;
}
.impact-card h3 {
  margin: 0 0 14px;
}
.impact-card > div {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14px;
}
.impact-card span,
.impact-card strong {
  display: block;
}
.impact-card span {
  color: var(--muted);
  font-size: 12px;
}
.impact-card strong {
  margin-top: 3px;
  color: var(--text);
  font-size: 20px;
}
.impact-card p {
  margin: 14px 0 0;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.6;
}
.drawer-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
.attachment-ready {
  color: #16803b;
}
.attachment-missing {
  color: #be2929;
}
.el-upload small {
  display: block;
  margin-top: 6px;
  color: var(--muted);
}
.el-select {
  width: 100%;
}
@media (max-width: 720px) {
  .admin-ticket-page {
    padding: 18px 12px;
  }
  .page-heading {
    align-items: stretch;
    flex-direction: column;
  }
  .summary-grid {
    grid-template-columns: 1fr;
  }
  .impact-card > div {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
