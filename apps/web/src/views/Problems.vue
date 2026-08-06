<template>
  <section class="page sub-page">
    <div class="sub-hero">
      <div class="sub-hero-inner">
        <div class="sub-hero-text">
          <h1 class="sub-hero-title">题库</h1>
          <p class="sub-hero-sub">{{ canAuthor ? '创建、维护并发布高质量编程题目' : '按难度、标签与完成状态查找练习' }}</p>
        </div>
        <div class="sub-hero-stats">
          <div class="sub-hero-stat"><span class="sub-hero-stat-val">{{ problems.length }}</span><span class="sub-hero-stat-label">题目总数</span></div>
          <div class="sub-hero-stat"><span class="sub-hero-stat-val">{{ tagOptions.length }}</span><span class="sub-hero-stat-label">标签数</span></div>
          <div v-if="auth.role === 'student'" class="sub-hero-stat"><span class="sub-hero-stat-val">{{ solvedCount }}</span><span class="sub-hero-stat-label">已解决</span></div>
        </div>
      </div>
    </div>

    <div class="sub-content">
      <div class="panel-header">
        <div class="toolbar">
          <el-button v-if="canAuthor" type="primary" @click="router.push('/problems/create')">新建题目</el-button>
          <el-button v-if="canAuthor" @click="uploadVisible = true">导入题目 ZIP</el-button>
          <el-button v-if="canPublishPrepared" @click="openPreparedPublish">从预备题库发布</el-button>
          <el-button @click="load">刷新</el-button>
        </div>
      </div>

      <div class="panel problem-filters">
        <el-input v-model="filters.keyword" clearable placeholder="搜索题号、标题或标签" />
        <ProblemTagSelector v-model="filters.tags" />
        <el-select v-model="filters.difficulty" clearable placeholder="难度">
          <el-option v-for="item in problemDifficultyOptions" :key="item" :label="item" :value="item">
            <div class="difficulty-option"><i :class="difficultyClass(item)"></i><span>{{ item }}</span></div>
          </el-option>
        </el-select>
        <el-select v-if="auth.role === 'student'" v-model="filters.status" placeholder="状态">
          <el-option v-for="option in problemStatusOptions" :key="option.value" :label="option.label" :value="option.value" />
        </el-select>
        <el-button @click="resetFilters">重置</el-button>
        <!-- <span class="muted filter-count">{{ filteredProblems.length }} / {{ problems.length }}</span> -->
      </div>

      <section class="panel problem-list-panel">
        <el-table :data="pagedProblems" row-key="id" @row-click="openProblem">
          <el-table-column label="题号" width="100">
            <template #default="{ row }"><span class="problem-code">{{ problemDisplayCode(row) }}</span></template>
          </el-table-column>
          <el-table-column label="题目名称" min-width="250">
            <template #default="{ row }">
              <router-link class="problem-title-link" :to="problemPath(row)" @click.stop>{{ row.title }}</router-link>
            </template>
          </el-table-column>
          <el-table-column label="难度" width="110">
            <template #default="{ row }">
              <el-tag
                :type="difficultyTagType(problemDifficulty(row))"
                :class="difficultyClass(problemDifficulty(row))"
                effect="light"
              >
                {{ problemDifficulty(row) || '未分级' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="标签" min-width="230">
            <template #default="{ row }">
              <div v-if="visibleTags(row).length" class="tag-strip table-tags">
                <el-tag v-for="tag in visibleTags(row)" :key="tag" size="small" effect="plain">{{ tag }}</el-tag>
              </div>
              <span v-else class="muted">-</span>
            </template>
          </el-table-column>
          <el-table-column label="运行限制" min-width="190">
            <template #default="{ row }">{{ problemLimitText(row) }}</template>
          </el-table-column>
          <el-table-column v-if="auth.role === 'student'" label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="progressTag(row.progress_status)" effect="light">{{ progressLabel(row.progress_status) }}</el-tag>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="!filteredProblems.length" description="暂无符合条件的题目" />
        <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="filteredProblems.length" />
      </section>
    </div>

    <el-dialog v-model="uploadVisible" title="导入题目包" width="560px">
      <el-upload drag :show-file-list="false" :http-request="upload" accept=".zip">
        <div class="upload-text">选择或拖入题目 ZIP 包</div>
        <div class="muted">题目内部编号由系统自动分配；支持 problem.yaml 题面、资源与测试点。</div>
      </el-upload>
      <template #footer><el-button @click="uploadVisible = false">关闭</el-button></template>
    </el-dialog>

    <el-dialog v-model="preparedPublishVisible" title="从预备题库发布" width="760px">
      <el-form label-width="90px">
        <el-form-item label="预备题">
          <el-select v-model="preparedIDs" multiple filterable style="width: 100%">
            <el-option
              v-for="item in preparedItems"
              :key="item.id"
              :label="preparedLabel(item)"
              :value="item.id"
              :disabled="Boolean(item.published_at)"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <el-table :data="preparedItems" height="260">
        <el-table-column label="题号" width="100">
          <template #default="{ row }">{{ problemDisplayCode(row.problem) }}</template>
        </el-table-column>
        <el-table-column prop="problem.title" label="题目" />
        <el-table-column prop="folder" label="文件夹" width="130" />
        <el-table-column prop="difficulty" label="难度" width="90" />
      </el-table>
      <template #footer>
        <el-button @click="preparedPublishVisible = false">取消</el-button>
        <el-button type="primary" :loading="publishingPrepared" @click="publishPrepared">立即公开</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { client, type PreparedProblem, type Problem } from '../api/client'
import ListPagination from '../components/ListPagination.vue'
import ProblemTagSelector from '../components/ProblemTagSelector.vue'
import {
  difficultyClass,
  difficultyTagType,
  problemDifficulty,
  problemDifficultyOptions,
  problemDisplayCode,
  problemLimitText,
  problemMatchesFilters,
  problemStatusOptions,
  progressLabel,
  progressTag,
  tagList,
  type ProblemFilters
} from '../features/problems/problemMeta'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const canAuthor = computed(() => Boolean(auth.user?.can_author) || auth.role === 'admin')
const canPublishPrepared = computed(() => auth.role === 'admin' || auth.role === 'teacher')
const problems = ref<Problem[]>([])
const filters = reactive<ProblemFilters>({ keyword: '', tags: [], difficulty: '', status: 'all' })
const uploadVisible = ref(false)
const preparedPublishVisible = ref(false)
const preparedItems = ref<PreparedProblem[]>([])
const preparedIDs = ref<number[]>([])
const publishingPrepared = ref(false)
const page = ref(1)
const pageSize = ref(20)
const tagOptions = computed(() => {
  const set = new Set(problems.value.flatMap(visibleTags))
  return [...set].sort((a, b) => a.localeCompare(b, 'zh-CN'))
})
const filteredProblems = computed(() => problems.value.filter((problem) => problemMatchesFilters(problem, filters)))
const pagedProblems = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredProblems.value.slice(start, start + pageSize.value)
})
const solvedCount = computed(() => problems.value.filter((problem) => problem.progress_status === 'accepted').length)

async function load() {
  try {
    problems.value = (await client.get('/problems')).data
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

function visibleTags(problem: Problem) {
  return tagList(problem.tags).filter((tag) => !problemDifficultyOptions.includes(tag as typeof problemDifficultyOptions[number]))
}

function problemPath(problem: Problem) {
  return `/problems/${encodeURIComponent(problem.display_code || String(problem.id))}`
}

function openProblem(problem: Problem) {
  router.push(problemPath(problem))
}

function resetFilters() {
  filters.keyword = ''
  filters.tags = []
  filters.difficulty = ''
  filters.status = 'all'
}

async function upload(options: any) {
  try {
    const form = new FormData()
    form.append('package', options.file)
    await client.post('/problems/upload', form)
    uploadVisible.value = false
    ElMessage.success('题目包已导入')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

async function openPreparedPublish() {
  preparedPublishVisible.value = true
  preparedIDs.value = []
  preparedItems.value = (await client.get('/prepared-problems')).data
}

function preparedLabel(item: PreparedProblem) {
  const suffix = [item.folder, item.difficulty, visibleTags(item.problem).join('/')].filter(Boolean).join(' · ')
  return `${problemDisplayCode(item.problem)}. ${item.problem?.title || '未知题目'}${suffix ? `（${suffix}）` : ''}`
}

async function publishPrepared() {
  if (!preparedIDs.value.length) {
    ElMessage.error('请选择预备题')
    return
  }
  publishingPrepared.value = true
  try {
    await Promise.all(preparedIDs.value.map((id) => client.post(`/prepared-problems/${id}/publish`)))
    preparedPublishVisible.value = false
    ElMessage.success('已发布到公共题库')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    publishingPrepared.value = false
  }
}

watch(
  () => [filters.keyword, filters.tags.join(','), filters.difficulty, filters.status, pageSize.value],
  () => { page.value = 1 }
)
onMounted(load)
</script>

<style scoped>
.sub-page { padding: 0; overflow-x: hidden; }
.sub-hero { background: linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0a5ea6 100%); }
.sub-hero-inner { max-width: 1200px; margin: 0 auto; padding: 32px 36px 40px; display: flex; flex-direction: column; gap: 20px; }
.sub-hero-title { margin: 0; color: #f8fafc; font-size: 28px; }
.sub-hero-sub { margin: 5px 0 0; color: rgba(248,250,252,.66); }
.sub-hero-stats { display: flex; flex-wrap: wrap; gap: 12px; }
.sub-hero-stat { min-width: 92px; display: grid; gap: 2px; padding: 10px 20px; text-align: center; border: 1px solid rgba(255,255,255,.15); border-radius: 10px; background: rgba(255,255,255,.1); }
.sub-hero-stat-val { color: #fff; font-size: 22px; font-weight: 700; }
.sub-hero-stat-label { color: rgba(248,250,252,.58); font-size: 12px; }
.sub-content { max-width: 1200px; margin: 0 auto; padding: 20px 20px 36px; }
.panel-header { display: flex; justify-content: flex-end; margin-bottom: 14px; }
.problem-filters { display: grid; grid-template-columns: minmax(220px, 1.5fr) minmax(150px, .8fr) minmax(130px, .7fr) minmax(130px, .7fr) auto auto; gap: 10px; margin-bottom: 14px; }
.filter-count { align-self: center; white-space: nowrap; }
.problem-list-panel { overflow: hidden; }
.problem-code { color: var(--accent); font-weight: 800; }
.problem-title-link { color: var(--text); font-weight: 700; }
.problem-title-link:hover { color: var(--accent); }
.tag-strip { display: flex; flex-wrap: wrap; gap: 6px; }
.difficulty-option { display: flex; align-items: center; gap: 9px; }
.difficulty-option i { width: 10px; height: 10px; border-radius: 50%; }
.difficulty-level-1 { --difficulty-color: #16a34a; }
.difficulty-level-2 { --difficulty-color: #0284c7; }
.difficulty-level-3 { --difficulty-color: #d97706; }
.difficulty-level-4 { --difficulty-color: #ea580c; }
.difficulty-level-5 { --difficulty-color: #dc2626; }
.difficulty-level-6 { --difficulty-color: #7c3aed; }
.difficulty-option i { background: var(--difficulty-color, #64748b); }
:deep(.el-tag.difficulty-level-1) { color: #15803d; border-color: #86efac; background: #f0fdf4; }
:deep(.el-tag.difficulty-level-2) { color: #0369a1; border-color: #7dd3fc; background: #f0f9ff; }
:deep(.el-tag.difficulty-level-3) { color: #b45309; border-color: #fcd34d; background: #fffbeb; }
:deep(.el-tag.difficulty-level-4) { color: #c2410c; border-color: #fdba74; background: #fff7ed; }
:deep(.el-tag.difficulty-level-5) { color: #b91c1c; border-color: #fca5a5; background: #fef2f2; }
:deep(.el-tag.difficulty-level-6) { color: #6d28d9; border-color: #c4b5fd; background: #f5f3ff; }
.upload-text { margin-bottom: 6px; font-weight: 700; }
@media (max-width: 900px) {
  .problem-filters { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 620px) {
  .sub-hero-inner { padding: 26px 18px 32px; }
  .sub-content { padding: 14px 12px 28px; }
  .problem-filters { grid-template-columns: 1fr; }
}
</style>
