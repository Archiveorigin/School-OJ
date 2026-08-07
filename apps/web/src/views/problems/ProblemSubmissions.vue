<template>
  <section class="page problem-records-page">
    <div class="records-container">
      <el-page-header title="返回题目" @back="goBack">
        <template #content>
          <span>{{ problem ? `${problemDisplayCode(problem)} · ${problem.title}` : '提交记录' }}</span>
        </template>
      </el-page-header>

      <header class="records-heading">
        <div>
          <span class="eyebrow">PUBLIC SOLUTIONS</span>
          <h1>题目提交记录</h1>
          <p>可查看自己的全部提交，以及其他用户主动公开的题库练习代码。</p>
        </div>
        <el-button type="primary" @click="goBack">返回提交代码</el-button>
      </header>

      <section class="panel record-filters">
        <el-select v-model="filters.language" clearable placeholder="代码语言">
          <el-option label="C++17" value="cpp" />
          <el-option label="C" value="c" />
          <el-option label="Python 3" value="python" />
          <el-option label="Java 21" value="java" />
        </el-select>
        <el-input
          v-model="filters.username"
          clearable
          placeholder="输入用户名查询"
          @keyup.enter="search"
        />
        <el-button type="primary" :loading="loading" @click="search">搜索</el-button>
        <el-button @click="resetFilters">清空筛选</el-button>
      </section>

      <section class="panel records-table">
        <el-table :data="pagedItems" v-loading="loading" @row-click="openDetail">
          <el-table-column label="提交人" min-width="150">
            <template #default="{ row }">{{ row.user_name || '未知用户' }}</template>
          </el-table-column>
          <el-table-column label="可见性" width="110">
            <template #default="{ row }">
              <el-tag :type="row.user_id === auth.user?.id ? 'info' : 'success'" effect="light">
                {{ row.user_id === auth.user?.id && !row.is_public ? '仅自己' : '公开' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="语言" width="110">
            <template #default="{ row }">{{ languageLabel(row.language) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="130">
            <template #default="{ row }"><StatusBadge :status="row.status" /></template>
          </el-table-column>
          <el-table-column prop="score" label="分数" width="90" />
          <el-table-column prop="time_ms" label="耗时 ms" width="100" />
          <el-table-column prop="memory_kb" label="内存 KB" width="110" />
          <el-table-column label="提交时间" min-width="180">
            <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button type="primary" text @click.stop="openDetail(row)">查看代码</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="!loading && !items.length" description="暂无符合条件的提交" />
        <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="items.length" />
      </section>
    </div>

    <SubmissionDetailDialog v-model="detailVisible" :detail="detail" />
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { onMounted, reactive, ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, type Problem, type Submission } from '../../api/client'
import ListPagination from '../../components/ListPagination.vue'
import SubmissionDetailDialog from '../../components/SubmissionDetailDialog.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { problemDisplayCode } from '../../features/problems/problemMeta'
import { formatDateTime } from '../../features/time'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const problem = ref<Problem | null>(null)
const items = ref<Submission[]>([])
const detail = ref<any>(null)
const detailVisible = ref(false)
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const filters = reactive({ language: '', username: '' })
const pagedItems = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return items.value.slice(start, start + pageSize.value)
})

async function loadProblem() {
  problem.value = (await client.get(`/problems/${encodeURIComponent(String(route.params.id))}`)).data
}

async function search() {
  if (!problem.value) return
  loading.value = true
  try {
    items.value = (await client.get('/submissions', {
      params: {
        visibility: 'problem',
        problem_id: problem.value.id,
        language: filters.language || undefined,
        username: filters.username.trim() || undefined
      }
    })).data
    page.value = 1
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

async function resetFilters() {
  filters.language = ''
  filters.username = ''
  await search()
}

async function openDetail(row: Submission) {
  try {
    detail.value = (await client.get(`/submissions/${row.id}`)).data
    detailVisible.value = true
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

function goBack() {
  router.push(`/problems/${encodeURIComponent(String(route.params.id))}`)
}

function languageLabel(language: string) {
  return ({ cpp: 'C++17', c: 'C', python: 'Python 3', java: 'Java 21' } as Record<string, string>)[language] || language
}

watch(pageSize, () => { page.value = 1 })
onMounted(async () => {
  try {
    await loadProblem()
    await search()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
})
</script>

<style scoped>
.problem-records-page { padding: 22px 20px 48px; }
.records-container { width: min(1180px, 100%); margin: 0 auto; }
.records-heading { display: flex; align-items: end; justify-content: space-between; gap: 24px; padding: 34px 0 24px; }
.records-heading h1 { margin: 6px 0; font-size: 30px; }
.records-heading p { margin: 0; color: var(--muted); }
.eyebrow { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .16em; }
.record-filters { display: grid; grid-template-columns: 180px minmax(220px, 1fr) auto auto; gap: 12px; margin-bottom: 16px; }
.records-table { overflow: hidden; }
@media (max-width: 760px) {
  .problem-records-page { padding: 16px 12px 36px; }
  .records-heading { align-items: stretch; flex-direction: column; }
  .record-filters { grid-template-columns: 1fr; }
}
</style>
