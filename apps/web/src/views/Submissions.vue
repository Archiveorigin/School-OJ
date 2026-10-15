<template>
  <section class="page sub-page">
    <div class="sub-hero">
      <div class="sub-hero-inner">
        <div class="sub-hero-stats">
          <div class="sub-hero-stat">
            <span class="sub-hero-stat-val">{{ items.length }}</span>
            <span class="sub-hero-stat-label">总提交</span>
          </div>
          <div class="sub-hero-stat">
            <span class="sub-hero-stat-val">{{ acceptedCount }}</span>
            <span class="sub-hero-stat-label">已通过</span>
          </div>
          <div class="sub-hero-stat">
            <span class="sub-hero-stat-val">{{ acceptRate }}%</span>
            <span class="sub-hero-stat-label">通过率</span>
          </div>
        </div>
      </div>
    </div>

    <div class="sub-content">
      <section class="panel submission-filters">
        <el-select v-model="filters.language" clearable placeholder="代码语言" aria-label="代码语言">
          <el-option label="C++17" value="cpp" /><el-option label="C" value="c" /><el-option label="Python 3" value="python" /><el-option label="Java 21" value="java" />
        </el-select>
        <el-input v-model="filters.username" clearable placeholder="输入用户名查询" @keyup.enter="load" />
        <el-checkbox v-model="filters.mine" @change="load">我的提交</el-checkbox>
        <el-button type="primary" :loading="loading" @click="load">搜索</el-button>
        <el-button @click="resetFilters">清空筛选</el-button>
      </section>
      <div class="panel">
        <el-table :data="pagedItems" v-loading="loading" @row-click="open">
          <el-table-column label="提交人" min-width="140">
            <template #default="{ row }">{{ submitterText(row) }}</template>
          </el-table-column>
          <el-table-column label="题目" min-width="220">
            <template #default="{ row }">{{ problemText(row) }}</template>
          </el-table-column>
          <el-table-column label="来源" min-width="140">
            <template #default="{ row }">{{ contextText(row) }}</template>
          </el-table-column>
          <el-table-column prop="language" label="语言" width="110" />
          <el-table-column label="状态" width="130">
            <template #default="{ row }"><StatusBadge :status="row.status" /></template>
          </el-table-column>
          <el-table-column prop="score" label="分数" width="90" />
          <el-table-column label="消息" min-width="260">
            <template #default="{ row }">
              <div class="message-preview">{{ row.message || '-' }}</div>
            </template>
          </el-table-column>
          <el-table-column label="时间" min-width="170">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </el-table-column>
        </el-table>
        <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="items.length" />
      </div>
    </div>

    <SubmissionDetailDialog v-model="visible" :detail="detail" />
  </section>
</template>

<script setup lang="ts">
import { apiErrorMessage } from '../api/errors'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { generatedApi, type Submission } from '../api/client'
import ListPagination from '../components/ListPagination.vue'
import SubmissionDetailDialog from '../components/SubmissionDetailDialog.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { formatDateTime } from '../features/time'

const items = ref<Submission[]>([])
const filters = reactive({ language: '', username: '', mine: false })
const loading = ref(false)
let pending: AbortController | undefined
onBeforeUnmount(() => pending?.abort())
const detail = ref<any>(null)
const visible = ref(false)
const page = ref(1)
const pageSize = ref(10)
const pagedItems = computed(() => items.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
const acceptedCount = computed(() => items.value.filter((s) => s.status === 'accepted').length)
const acceptRate = computed(() => {
  if (items.value.length === 0) return 0
  return Math.round((acceptedCount.value / items.value.length) * 100)
})

async function load() {
  pending?.abort()
  const request = new AbortController()
  pending = request
  loading.value = true
  try {
    const { data } = await generatedApi.listSubmissions<Submission[]>({ signal: request.signal, query: { language: filters.language || undefined, username: filters.username.trim() || undefined, mine: filters.mine || undefined } })
    if (request.signal.aborted) return
    items.value = data
    page.value = 1
  } catch (err: any) {
    if (!request.signal.aborted) ElMessage.error(apiErrorMessage(err))
  } finally {
    if (pending === request) loading.value = false
  }
}
function resetFilters() {
  Object.assign(filters, { language: '', username: '', mine: false })
  void load()
}

async function open(row: Submission) {
  detail.value = (await generatedApi.getSubmission<any>({ path: { "id": row.id } })).data
  visible.value = true
}

function submitterText(row: Submission) {
  return row.user_name ? `${row.user_name}${row.student_no ? `（${row.student_no}）` : ''}` : '-'
}

function problemText(row: Submission) {
  const code = row.problem_code || '未编号'
  return row.problem_title ? `${code} · ${row.problem_title}` : code
}

function contextText(row: Submission) {
  if (row.exam_title) return `考试：${row.exam_title}`
  if (row.assignment_title) return `作业：${row.assignment_title}`
  return '题库练习'
}

onMounted(load)
watch(pageSize, clampPage)

function clampPage() {
  const maxPage = Math.max(1, Math.ceil(items.value.length / pageSize.value))
  if (page.value > maxPage) page.value = maxPage
  if (page.value < 1) page.value = 1
}
</script>

<style scoped>
.submission-filters { display: grid; grid-template-columns: 180px minmax(180px, 1fr) auto auto auto; gap: 12px; align-items: center; margin-bottom: 16px; }
.sub-page {
  padding: 0;
  overflow-x: hidden;
}

.sub-hero {
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0a5ea6 100%);
}

.sub-hero-inner {
  max-width: 1200px;
  margin: 0 auto;
  padding: 32px 36px 40px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.sub-hero-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.sub-hero-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 20px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 10px;
  min-width: 80px;
  text-align: center;
  transition: background 0.2s;
}

.sub-hero-stat:hover {
  background: rgba(255, 255, 255, 0.18);
}

.sub-hero-stat-val {
  font-size: 22px;
  font-weight: 700;
  color: #fff;
}

.sub-hero-stat-label {
  font-size: 12px;
  color: rgba(248, 250, 252, 0.55);
}

.sub-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px 20px 32px;
}

.message-preview {
  display: -webkit-box;
  max-height: 42px;
  overflow: hidden;
  color: #4b5563;
  line-height: 1.35;
  word-break: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

@media (max-width: 760px) {
  .submission-filters { grid-template-columns: 1fr; }
  .sub-hero-inner {
    padding: 24px 20px 32px;
    gap: 16px;
  }
}
</style>
