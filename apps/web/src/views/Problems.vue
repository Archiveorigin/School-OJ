<template>
  <section class="page sub-page">
    <div class="sub-hero">
      <div class="sub-hero-inner">
        <div class="sub-hero-text">
          <h1 class="sub-hero-title">题库</h1>
          <p class="sub-hero-sub">{{ canManage ? '管理公共题库、上传题目包、发布预备题' : '浏览公共题库并进入独立题目页面' }}</p>
        </div>
        <div class="sub-hero-stats">
          <div class="sub-hero-stat">
            <span class="sub-hero-stat-val">{{ problems.length }}</span>
            <span class="sub-hero-stat-label">题目总数</span>
          </div>
          <div class="sub-hero-stat">
            <span class="sub-hero-stat-val">{{ tagOptions.length }}</span>
            <span class="sub-hero-stat-label">标签数</span>
          </div>
          <div v-if="auth.isAuthed && !canManage" class="sub-hero-stat">
            <span class="sub-hero-stat-val">{{ solvedCount }}</span>
            <span class="sub-hero-stat-label">已解决</span>
          </div>
        </div>
      </div>
    </div>

    <div class="sub-content">
      <div class="panel-header">
        <div class="toolbar">
          <el-button v-if="canManage" type="primary" @click="openProblemDialog">上传题目包</el-button>
          <el-button v-if="canManage" @click="openPreparedPublish">从预备题库发布</el-button>
          <el-button @click="load">刷新</el-button>
        </div>
      </div>

      <div class="panel problem-filters">
        <el-input v-model="filters.keyword" clearable placeholder="搜索编号、标题、Slug、标签" />
        <el-select v-model="filters.tag" clearable filterable placeholder="标签">
          <el-option v-for="tag in tagOptions" :key="tag" :label="tag" :value="tag" />
        </el-select>
        <el-select v-model="filters.difficulty" clearable placeholder="难度">
          <el-option v-for="item in difficultyOptions" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select v-if="auth.role === 'student'" v-model="filters.status" placeholder="状态">
          <el-option
            v-for="option in problemStatusOptions"
            :key="option.value"
            :label="option.label"
            :value="option.value"
          />
        </el-select>
        <el-button @click="resetFilters">重置</el-button>
        <span class="muted filter-count">{{ filteredProblems.length }} / {{ problems.length }}</span>
      </div>

      <section class="panel problem-list-panel">
        <el-table :data="pagedProblems" row-key="id" @row-click="openProblem">
          <el-table-column label="题号" width="100">
            <template #default="{ row }">
              <span class="problem-code">{{ problemDisplayCode(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="题目名称" min-width="240">
            <template #default="{ row }">
              <router-link class="problem-title-link" :to="problemPath(row)" @click.stop>
                {{ row.title }}
              </router-link>
              <div class="muted problem-slug">{{ row.slug }}</div>
            </template>
          </el-table-column>
          <el-table-column label="难度" width="110">
            <template #default="{ row }">
              <el-tag :type="difficultyTagType(problemDifficulty(row))" effect="light">
                {{ problemDifficulty(row) || '未分级' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="标签" min-width="220">
            <template #default="{ row }">
              <div v-if="tagList(row.tags).length" class="tag-strip table-tags">
                <el-tag v-for="tag in tagList(row.tags)" :key="tag" size="small" effect="plain">
                  {{ tag }}
                </el-tag>
              </div>
              <span v-else class="muted">-</span>
            </template>
          </el-table-column>
          <el-table-column label="运行限制" min-width="180">
            <template #default="{ row }">{{ problemLimitText(row) }}</template>
          </el-table-column>
          <el-table-column v-if="auth.role === 'student'" label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="progressTag(row.progress_status)" effect="light">
                {{ progressLabel(row.progress_status) }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="!filteredProblems.length" description="暂无符合条件的题目" />
        <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="filteredProblems.length" />
      </section>
    </div>

    <el-dialog v-model="problemDialogVisible" title="上传题目包" width="920px">
      <el-tabs v-model="problemDialogTab">
        <el-tab-pane label="上传现有 ZIP" name="zip">
          <el-upload
            drag
            :show-file-list="false"
            :http-request="upload"
            accept=".zip"
            class="zip-upload"
          >
            <div class="upload-text">选择或拖入题目包 ZIP</div>
            <div class="muted">problem.yaml 的 statement 可使用 Markdown 和 LaTeX 多行文本</div>
          </el-upload>
        </el-tab-pane>
        <el-tab-pane label="表单创建题目" name="form">
          <el-form label-width="110px" class="problem-form">
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="Slug">
                  <el-input v-model="problemForm.slug" placeholder="two-sum" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="标题">
                  <el-input v-model="problemForm.title" placeholder="两数之和" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-form-item label="题面">
              <el-input
                v-model="problemForm.statement"
                type="textarea"
                :rows="8"
                placeholder="支持 Markdown 和 LaTeX"
              />
              <div class="statement-tools">
                <el-upload
                  action="#"
                  :auto-upload="false"
                  :show-file-list="false"
                  multiple
                  accept="image/png,image/jpeg,image/gif,image/webp"
                  :on-change="addProblemImage"
                >
                  <el-button>插入图片</el-button>
                </el-upload>
                <span class="muted">支持 PNG、JPG、GIF、WebP</span>
              </div>
              <div v-if="problemForm.assets.length" class="asset-row">
                <el-tag v-for="asset in problemForm.assets" :key="asset.path" closable @close="removeProblemImage(asset.path)">
                  {{ asset.name }}
                </el-tag>
              </div>
              <div class="statement-preview">
                <div class="muted">题面预览</div>
                <MarkdownRenderer :source="problemForm.statement || '支持 **Markdown** 和 $a+b$。'" :asset-urls="problemAssetPreviewUrls" />
              </div>
            </el-form-item>
            <el-row :gutter="12">
              <el-col :span="8">
                <el-form-item label="时间限制">
                  <el-input-number v-model="problemForm.time_limit_ms" :min="100" :step="100" />
                  <span class="unit">ms</span>
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="内存限制">
                  <el-input-number v-model="problemForm.memory_limit_mb" :min="16" :step="16" />
                  <span class="unit">MB</span>
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="输出限制">
                  <el-input-number v-model="problemForm.output_limit_kb" :min="1" :step="64" />
                  <span class="unit">KB</span>
                </el-form-item>
              </el-col>
            </el-row>
            <el-form-item label="前台样例">
              <ProblemSampleEditor v-model="problemForm.samples" />
            </el-form-item>
            <el-form-item label="测试点文件">
              <div class="test-file-upload">
                <el-upload
                  drag
                  action="#"
                  multiple
                  accept=".zip,.in,.out"
                  :auto-upload="false"
                  :file-list="problemTestFiles"
                  :on-change="syncProblemTestFiles"
                  :on-remove="syncProblemTestFiles"
                >
                  <div class="upload-text">选择或拖入 .zip / .in / .out 测试点文件</div>
                  <div class="muted">文件名需含数字序号；上传文件后将以文件内容作为后台评测点。</div>
                </el-upload>
              </div>
            </el-form-item>
            <div class="case-toolbar">
              <div>
                <h4>手动填写后台测试点</h4>
                <span class="muted">未上传测试点文件时使用下方数据</span>
              </div>
              <el-button size="small" @click="addCase">添加测试点</el-button>
            </div>
            <div v-for="(item, index) in problemForm.cases" :key="index" class="case-editor">
              <div class="case-head">
                <el-input v-model="item.name" placeholder="测试点名称" />
                <el-input-number v-model="item.weight" :min="1" :max="100" />
                <el-button :disabled="problemForm.cases.length === 1" @click="removeCase(index)">
                  删除
                </el-button>
              </div>
              <el-row :gutter="12">
                <el-col :span="12">
                  <el-input v-model="item.input" type="textarea" :rows="5" placeholder="输入数据" />
                </el-col>
                <el-col :span="12">
                  <el-input v-model="item.output" type="textarea" :rows="5" placeholder="期望输出" />
                </el-col>
              </el-row>
            </div>
          </el-form>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button @click="problemDialogVisible = false">取消</el-button>
        <el-button
          v-if="problemDialogTab === 'form'"
          type="primary"
          :loading="savingProblem"
          @click="createFromForm"
        >
          创建题目
        </el-button>
      </template>
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
        <el-table-column prop="problem.slug" label="Slug" width="140" />
        <el-table-column prop="problem.title" label="题目" />
        <el-table-column prop="folder" label="文件夹" width="130" />
        <el-table-column prop="difficulty" label="难度" width="90" />
      </el-table>
      <template #footer>
        <el-button @click="preparedPublishVisible = false">取消</el-button>
        <el-button type="primary" :loading="publishingPrepared" @click="publishPrepared">
          立即公开
        </el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { client, type PreparedProblem, type Problem } from '../api/client'
import MarkdownRenderer from '../components/MarkdownRenderer.vue'
import ListPagination from '../components/ListPagination.vue'
import ProblemSampleEditor from '../components/ProblemSampleEditor.vue'
import {
  difficultyFromTags,
  difficultyTagType,
  problemDisplayCode,
  problemLimitText,
  problemMatchesFilters,
  problemStatusOptions,
  progressLabel,
  progressTag,
  replaceStatementSamples,
  tagList,
  type ProblemFilters
} from '../features/problems/problemMeta'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
type ProblemAssetForm = { name: string; path: string; content_type: string; data: string; preview_url: string }
const problems = ref<Problem[]>([])
const filters = reactive<ProblemFilters>({
  keyword: '',
  tag: '',
  difficulty: '',
  status: 'all'
})
const problemDialogVisible = ref(false)
const problemDialogTab = ref('zip')
const savingProblem = ref(false)
const problemTestFiles = ref<any[]>([])
const preparedPublishVisible = ref(false)
const preparedItems = ref<PreparedProblem[]>([])
const preparedIDs = ref<number[]>([])
const publishingPrepared = ref(false)
const page = ref(1)
const pageSize = ref(20)
const problemForm = reactive({
  slug: '',
  title: '',
  statement: '',
  time_limit_ms: 1000,
  memory_limit_mb: 256,
  output_limit_kb: 1024,
  assets: [] as ProblemAssetForm[],
  samples: [] as Array<{ name: string; input: string; output: string }>,
  cases: [{ name: 'case-01', input: '1 2\n', output: '3\n', weight: 100 }]
})
const problemAssetPreviewUrls = computed(() => {
  return Object.fromEntries(problemForm.assets.map((asset) => [asset.path, asset.preview_url]))
})
const tagOptions = computed(() => {
  const set = new Set(problems.value.flatMap((problem) => tagList(problem.tags)))
  return [...set].sort((a, b) => a.localeCompare(b, 'zh-CN'))
})
const difficultyOptions = computed(() => {
  const set = new Set(problems.value.map(problemDifficulty).filter(Boolean))
  return [...set].sort((a, b) => a.localeCompare(b, 'zh-CN'))
})
const filteredProblems = computed(() => problems.value.filter((problem) => problemMatchesFilters(problem, filters)))
const pagedProblems = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredProblems.value.slice(start, start + pageSize.value)
})
const solvedCount = computed(() => problems.value.filter((p) => p.progress_status === 'accepted').length)

async function load() {
  problems.value = (await client.get('/problems')).data
}

function problemPath(problem: Problem) {
  return `/problems/${encodeURIComponent(problem.display_code || String(problem.id))}`
}

function openProblem(problem: Problem) {
  router.push(problemPath(problem))
}

function problemDifficulty(problem: Problem) {
  return difficultyFromTags(problem.tags)
}

function resetFilters() {
  filters.keyword = ''
  filters.tag = ''
  filters.difficulty = ''
  filters.status = 'all'
}

watch(
  () => [filters.keyword, filters.tag, filters.difficulty, filters.status, pageSize.value],
  () => { page.value = 1 }
)

function openProblemDialog() {
  problemDialogVisible.value = true
  problemDialogTab.value = 'zip'
}

async function openPreparedPublish() {
  preparedPublishVisible.value = true
  preparedIDs.value = []
  preparedItems.value = (await client.get('/prepared-problems')).data
}

function preparedLabel(item: PreparedProblem) {
  const tags = tagList(item.problem?.tags)
  const suffix = [item.folder, item.difficulty, tags.join('/')].filter(Boolean).join(' · ')
  const code = item.problem ? problemDisplayCode(item.problem) : '未编号'
  return `${code}. ${item.problem?.title || '未知题目'}${suffix ? `（${suffix}）` : ''}`
}

async function publishPrepared() {
  if (preparedIDs.value.length === 0) {
    ElMessage.error('请选择预备题')
    return
  }
  publishingPrepared.value = true
  try {
    await Promise.all(
      preparedIDs.value.map((id) =>
        client.post(`/prepared-problems/${id}/publish`)
      )
    )
    ElMessage.success('已发布到公共题库')
    preparedPublishVisible.value = false
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    publishingPrepared.value = false
  }
}

async function upload(options: any) {
  try {
    const fd = new FormData()
    fd.append('package', options.file)
    await client.post('/problems/upload', fd)
    ElMessage.success('题目包已上传')
    problemDialogVisible.value = false
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

function addCase() {
  const next = problemForm.cases.length + 1
  problemForm.cases.push({ name: `case-${String(next).padStart(2, '0')}`, input: '', output: '', weight: 1 })
}

function removeCase(index: number) {
  problemForm.cases.splice(index, 1)
}

async function createFromForm() {
  savingProblem.value = true
  try {
    const draft = {
      slug: problemForm.slug,
      title: problemForm.title,
      statement: replaceStatementSamples(problemForm.statement, problemForm.samples),
      time_limit_ms: problemForm.time_limit_ms,
      memory_limit_mb: problemForm.memory_limit_mb,
      output_limit_kb: problemForm.output_limit_kb,
      assets: problemForm.assets.map(({ name, path, content_type, data }) => ({ name, path, content_type, data })),
      cases: problemForm.cases
    }
    if (problemTestFiles.value.length) {
      const fd = new FormData()
      fd.append('draft', JSON.stringify(draft))
      for (const item of problemTestFiles.value) {
        if (item.raw) fd.append('test_files', item.raw)
      }
      await client.post('/problems', fd)
    } else {
      await client.post('/problems', draft)
    }
    ElMessage.success('题目已创建')
    problemDialogVisible.value = false
    await load()
    resetProblemForm()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    savingProblem.value = false
  }
}

function resetProblemForm() {
  problemForm.assets.forEach((asset) => URL.revokeObjectURL(asset.preview_url))
  problemForm.slug = ''
  problemForm.title = ''
  problemForm.statement = ''
  problemForm.time_limit_ms = 1000
  problemForm.memory_limit_mb = 256
  problemForm.output_limit_kb = 1024
  problemForm.assets.splice(0, problemForm.assets.length)
  problemForm.samples.splice(0, problemForm.samples.length)
  problemTestFiles.value = []
  problemForm.cases.splice(0, problemForm.cases.length, {
    name: 'case-01',
    input: '1 2\n',
    output: '3\n',
    weight: 100
  })
}

function syncProblemTestFiles(_file: any, fileList: any[]) {
  problemTestFiles.value = fileList
}

function addProblemImage(uploadFile: any) {
  const file = uploadFile.raw as File | undefined
  if (!file) return
  if (!['image/png', 'image/jpeg', 'image/gif', 'image/webp'].includes(file.type)) {
    ElMessage.error('仅支持 PNG、JPG、GIF、WebP 图片')
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    ElMessage.error('单张图片不能超过 5 MB')
    return
  }
  const path = uniqueAssetPath(file.name)
  const reader = new FileReader()
  reader.onload = () => {
    problemForm.assets.push({
      name: file.name,
      path,
      content_type: file.type,
      data: String(reader.result),
      preview_url: URL.createObjectURL(file)
    })
    const markdown = `![${file.name}](${path})`
    problemForm.statement = `${problemForm.statement.trimEnd()}\n\n${markdown}\n`
  }
  reader.readAsDataURL(file)
}

function removeProblemImage(path: string) {
  const index = problemForm.assets.findIndex((asset) => asset.path === path)
  if (index < 0) return
  URL.revokeObjectURL(problemForm.assets[index].preview_url)
  problemForm.assets.splice(index, 1)
  problemForm.statement = problemForm.statement.replace(new RegExp(`!\\[[^\\]]*\\]\\(${escapeRegExp(path)}\\)\\n?`, 'g'), '').trimEnd()
}

function uniqueAssetPath(name: string) {
  const safe = name
    .trim()
    .replace(/\s+/g, '-')
    .replace(/[^A-Za-z0-9._-]/g, '')
    .replace(/^\.+/, '')
  const fallback = `image-${Date.now()}.png`
  const base = safe || fallback
  let path = `assets/${base}`
  let index = 1
  while (problemForm.assets.some((asset) => asset.path === path)) {
    const dot = base.lastIndexOf('.')
    path = dot > 0 ? `assets/${base.slice(0, dot)}-${index}${base.slice(dot)}` : `assets/${base}-${index}`
    index += 1
  }
  return path
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

onMounted(load)
</script>

<style scoped>
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

.sub-hero-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sub-hero-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
  color: #f8fafc;
}

.sub-hero-sub {
  margin: 0;
  font-size: 14px;
  color: rgba(248, 250, 252, 0.6);
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

.panel-header {
  margin-bottom: 14px;
}

.problem-filters {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) 150px 130px minmax(120px, 140px) auto auto;
  gap: 10px;
  align-items: center;
  margin-bottom: 16px;
}

.tag-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 6px;
}

.filter-count {
  justify-self: end;
  white-space: nowrap;
}

.problem-list-panel {
  min-height: 320px;
  padding: 8px 12px 14px;
}

.problem-list-panel :deep(.el-table__row) {
  cursor: pointer;
}

.problem-code {
  color: var(--muted);
  font-variant-numeric: tabular-nums;
  font-weight: 700;
}

.problem-title-link {
  color: var(--text);
  font-weight: 700;
}

.problem-title-link:hover {
  color: var(--accent);
}

.problem-slug {
  margin-top: 3px;
  font-size: 12px;
}

.table-tags {
  margin-top: 0;
}

.zip-upload {
  width: 100%;
}

.upload-text {
  font-weight: 600;
  margin-bottom: 6px;
}

.problem-form {
  max-height: 60vh;
  overflow: auto;
  padding-right: 8px;
}

.unit {
  margin-left: 8px;
  color: #6b7280;
}

.case-toolbar,
.case-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.case-toolbar {
  margin: 8px 0 10px;
}

.case-toolbar h4 {
  margin: 0;
}

.case-toolbar .muted {
  display: block;
  margin-top: 3px;
  font-size: 12px;
}

.test-file-upload {
  width: 100%;
}

.case-editor {
  border: 1px solid #d9dee8;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
}

.case-head {
  margin-bottom: 10px;
}

.statement-preview {
  width: 100%;
  margin-top: 10px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.03);
}

.statement-tools {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  width: 100%;
  margin-top: 10px;
}

.asset-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  width: 100%;
  margin-top: 8px;
}

@media (max-width: 760px) {
  .sub-hero-inner {
    padding: 24px 20px 32px;
    gap: 16px;
  }

  .problem-filters {
    grid-template-columns: 1fr;
  }

  .filter-count {
    justify-self: start;
  }
}
</style>
