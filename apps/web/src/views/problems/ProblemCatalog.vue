<template>
  <section class="catalog-page">
    <header class="catalog-header">
      <div>
        <span class="eyebrow">PROBLEM LIBRARY</span>
        <h1>题库</h1>
        <p>按题号、难度与算法标签筛选，选择一道题开始练习。</p>
      </div>
      <el-button v-if="canAuthor" type="primary" @click="router.push('/problem-changes/new')">发起工单</el-button>
    </header>

    <section class="filter-card panel">
      <div class="filter-grid">
        <el-input v-model="filters.q" clearable placeholder="搜索题号或题目名称" aria-label="搜索题号或题目名称" />
        <el-select v-model="filters.difficulty" clearable placeholder="全部难度">
          <el-option v-for="item in problemDifficultyOptions" :key="item" :label="item" :value="item" />
        </el-select>
        <ProblemTagSelector v-model="filters.tags" />
        <el-select v-if="auth.isAuthed" v-model="filters.status" placeholder="完成状态">
          <el-option label="全部状态" value="" />
          <el-option label="未尝试" value="unattempted" />
          <el-option label="已尝试" value="attempted" />
          <el-option label="已通过" value="accepted" />
        </el-select>
        <el-button @click="resetFilters">重置</el-button>
      </div>
      <div class="active-filters">
        <span>已选筛选</span>
        <el-tag v-if="filters.difficulty" closable @close="filters.difficulty = ''">{{ filters.difficulty }}</el-tag>
        <el-tag v-for="tag in filters.tags" :key="tag" closable @close="removeTag(tag)">{{ tag }}</el-tag>
        <el-tag v-if="filters.status" type="info" closable @close="filters.status = ''">{{ statusLabel(filters.status) }}</el-tag>
        <small>{{ total }} 道结果</small>
      </div>
    </section>

    <section class="problem-panel panel" v-loading="loading">
      <el-table class="desktop-table" :data="items" row-key="id" @row-click="openProblem">
        <el-table-column label="状态" width="72" align="center">
          <template #default="{ row }"><span class="status-dot" :class="row.progress_status" :title="statusLabel(row.progress_status)"><el-icon><CircleCheckFilled v-if="row.progress_status === 'accepted'" /><Clock v-else-if="row.progress_status === 'attempted'" /><Minus v-else /></el-icon></span></template>
        </el-table-column>
        <el-table-column label="题号" width="100">
          <template #default="{ row }"><strong class="problem-code">{{ problemDisplayCode(row) }}</strong></template>
        </el-table-column>
        <el-table-column label="题目" min-width="260">
          <template #default="{ row }">
            <router-link class="problem-link" :to="problemPath(row)" @click.stop>{{ row.title }}</router-link>
          </template>
        </el-table-column>
        <el-table-column label="标签" min-width="250">
          <template #default="{ row }"><div class="tag-line"><el-tag v-for="tag in visibleTags(row).slice(0, 4)" :key="tag" size="small" effect="plain">{{ tag }}</el-tag><span v-if="!visibleTags(row).length" class="muted">—</span></div></template>
        </el-table-column>
        <el-table-column label="难度" width="105">
          <template #default="{ row }"><span class="difficulty" :class="difficultyClass(problemDifficulty(row))">{{ problemDifficulty(row) }}</span></template>
        </el-table-column>
        <el-table-column label="通过率" width="118" align="right">
          <template #default="{ row }"><span class="pass-rate">{{ passRate(row) }}</span><small>{{ row.accepted_count }}/{{ row.evaluated_count }}</small></template>
        </el-table-column>
      </el-table>

      <div class="mobile-list">
        <article v-for="row in items" :key="row.id" class="problem-card" @click="openProblem(row)">
          <div class="card-top"><strong>{{ problemDisplayCode(row) }}</strong><span class="status-dot" :class="row.progress_status"><el-icon><CircleCheckFilled v-if="row.progress_status === 'accepted'" /><Clock v-else-if="row.progress_status === 'attempted'" /><Minus v-else /></el-icon></span></div>
          <h2>{{ row.title }}</h2>
          <div class="tag-line"><el-tag v-for="tag in visibleTags(row).slice(0, 3)" :key="tag" size="small" effect="plain">{{ tag }}</el-tag></div>
          <footer><span class="difficulty" :class="difficultyClass(problemDifficulty(row))">{{ problemDifficulty(row) }}</span><span>{{ passRate(row) }} 通过</span></footer>
        </article>
      </div>
      <el-empty v-if="!loading && !items.length" description="暂无符合条件的题目" />
      <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="total" />
    </section>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { CircleCheckFilled, Clock, Minus } from '@element-plus/icons-vue'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { client, type Problem, type ProblemCatalogItem, type ProblemCatalogResponse } from '../../api/client'
import ListPagination from '../../components/ListPagination.vue'
import ProblemTagSelector from '../../components/ProblemTagSelector.vue'
import { difficultyClass, problemDifficulty, problemDifficultyOptions, problemDisplayCode, tagList } from '../../features/problems/problemMeta'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const canAuthor = computed(() => Boolean(auth.user?.can_author) || auth.role === 'admin')
const items = ref<ProblemCatalogItem[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const filters = reactive({ q: '', difficulty: '', tags: [] as string[], status: '' })
let loadTimer: number | undefined

async function load() {
  loading.value = true
  try {
    const { data } = await client.get<ProblemCatalogResponse>('/problems/catalog', {
      params: { page: page.value, page_size: pageSize.value, q: filters.q.trim() || undefined, difficulty: filters.difficulty || undefined, tags: filters.tags.join(',') || undefined, status: filters.status || undefined }
    })
    items.value = data.items || []
    total.value = data.total || 0
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || error.message)
  } finally {
    loading.value = false
  }
}

function scheduleLoad(resetPage = true) {
  if (resetPage) page.value = 1
  if (loadTimer) window.clearTimeout(loadTimer)
  loadTimer = window.setTimeout(load, 180)
}
function resetFilters() { Object.assign(filters, { q: '', difficulty: '', tags: [], status: '' }) }
function removeTag(tag: string) { filters.tags = filters.tags.filter((item) => item !== tag) }
function visibleTags(problem: Problem) { return tagList(problem.tags).filter((tag) => !problemDifficultyOptions.includes(tag as any)) }
function problemPath(problem: Problem) { return `/problems/${encodeURIComponent(problem.display_code || String(problem.id))}` }
function openProblem(problem: Problem) { void router.push(problemPath(problem)) }
function passRate(problem: ProblemCatalogItem) { return problem.evaluated_count ? `${problem.pass_rate.toFixed(1)}%` : '—' }
function statusLabel(status?: string) { return status === 'accepted' ? '已通过' : status === 'attempted' ? '已尝试' : status === 'unattempted' ? '未尝试' : '全部状态' }

watch(() => [filters.q, filters.difficulty, filters.tags.join(','), filters.status], () => scheduleLoad())
watch([page, pageSize], () => scheduleLoad(false))
onMounted(load)
onBeforeUnmount(() => { if (loadTimer) window.clearTimeout(loadTimer) })
</script>

<style scoped>
.catalog-page { width: min(1240px, calc(100% - 32px)); margin: 0 auto; padding: 30px 0 54px; }
.catalog-header { display: flex; align-items: flex-end; justify-content: space-between; gap: 22px; margin-bottom: 20px; }
.catalog-header h1 { margin: 6px 0 4px; font-size: 30px; }
.catalog-header p { margin: 0; color: var(--muted); }
.eyebrow { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .16em; }
.filter-card { margin-bottom: 14px; padding: 18px; }
.filter-grid { display: grid; grid-template-columns: minmax(220px, 1.5fr) repeat(3, minmax(140px, .85fr)) auto; gap: 10px; }
.active-filters { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; min-height: 28px; margin-top: 14px; padding-top: 14px; border-top: 1px solid var(--border); }
.active-filters > span { color: var(--muted); font-size: 13px; }
.active-filters small { margin-left: auto; color: var(--muted); }
.problem-panel { overflow: hidden; }
.problem-code { color: var(--accent); }
.problem-link { color: var(--text); font-weight: 700; }
.problem-link:hover { color: var(--accent); }
.tag-line { display: flex; flex-wrap: wrap; gap: 5px; }
.difficulty { font-weight: 700; }
.difficulty-level-1 { color: #16803b; }.difficulty-level-2 { color: #0374a8; }.difficulty-level-3 { color: #b56c09; }.difficulty-level-4 { color: #c84b16; }.difficulty-level-5 { color: #be2929; }.difficulty-level-6 { color: #7138b9; }
.pass-rate { display: block; color: var(--text); font-weight: 700; }.pass-rate + small { display: block; color: var(--muted); }
.status-dot { display: inline-grid; width: 24px; height: 24px; place-items: center; color: #94a3b8; border: 1px solid #cbd5e1; border-radius: 50%; font-weight: 900; }
.status-dot.accepted { color: #16803b; border-color: #86d39f; background: #eefbf2; }.status-dot.attempted { color: #d97706; border-color: #f4c66c; background: #fff8e8; }
.mobile-list { display: none; }
@media (max-width: 900px) { .filter-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }.filter-grid .el-button { width: 100%; }.active-filters small { width: 100%; margin-left: 0; } }
@media (max-width: 640px) {
  .catalog-page { width: min(100% - 24px, 1240px); padding-top: 20px; }.catalog-header { align-items: stretch; flex-direction: column; }.catalog-header .el-button { width: 100%; }.filter-grid { grid-template-columns: 1fr; }
  .desktop-table { display: none; }.mobile-list { display: grid; gap: 10px; padding: 12px; }.problem-card { padding: 15px; border: 1px solid var(--border); border-radius: 9px; background: var(--surface-strong); cursor: pointer; }.card-top, .problem-card footer { display: flex; align-items: center; justify-content: space-between; gap: 10px; }.card-top strong { color: var(--accent); }.problem-card h2 { margin: 10px 0; font-size: 17px; }.problem-card footer { margin-top: 14px; padding-top: 10px; color: var(--muted); border-top: 1px solid var(--border); font-size: 13px; }
}
</style>
